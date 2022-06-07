package charger

// LICENSE

// Copyright (c) 2019-2022 andig

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
)

// https://github.com/RustyDust/sonnen-charger/blob/main/Etrel%20INCH%20SmartHome%20Modbus%20TCPRegisters.xlsx

const (
	etrelRegChargeStatus  = 0
	etrelRegPower         = 26
	etrelRegSessionEnergy = 30
	etrelRegChargeTime    = 32
	etrelRegSerial        = 990
	etrelRegModel         = 1000
	etrelRegBrand         = 190
	etrelRegHWVersion     = 1010
	etrelRegSWVersion     = 1015

	etrelRegStop       = 1
	etrelRegPause      = 2
	etrelRegMaxCurrent = 8
)

var etrelRegCurrents = []uint16{14, 16, 18}

// Etrel is an api.Charger implementation for Etrel/Sonnen wallboxes
type Etrel struct {
	log     *util.Logger
	conn    *modbus.Connection
	current float32
}

func init() {
	registry.Add("etrel", NewEtrelFromConfig)
}

// NewEtrelFromConfig creates a Etrel charger from generic config
func NewEtrelFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEtrel(cc.URI, cc.ID)
}

// NewEtrel creates a Etrel charger
func NewEtrel(uri string, id uint8) (*Etrel, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	// if !sponsor.IsAuthorized() {
	// 	return nil, api.ErrSponsorRequired
	// }

	log := util.NewLogger("etrel")
	conn.Logger(log.TRACE)

	wb := &Etrel{
		log:     log,
		conn:    conn,
		current: 6,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Etrel) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegChargeStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch u := binary.BigEndian.Uint16(b); u {
	case 1, 2:
		return api.StatusA, nil
	case 3, 5, 6, 7, 9:
		return api.StatusB, nil
	case 4:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *Etrel) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(etrelRegMaxCurrent, 2)
	if err != nil {
		return false, err
	}

	return encoding.Float32(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Etrel) Enable(enable bool) error {
	if enable {
		return wb.setCurrent(wb.current)
	}

	b := make([]byte, 1)
	binary.BigEndian.PutUint16(b, 1)

	_, err := wb.conn.WriteMultipleRegisters(etrelRegStop, 1, b)
	return err
}

func (wb *Etrel) setCurrent(current float32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, math.Float32bits(wb.current))

	_, err := wb.conn.WriteMultipleRegisters(etrelRegMaxCurrent, 1, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Etrel) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Etrel)(nil)

// MaxCurrentMilli implements the api.ChargerEx interface
func (wb *Etrel) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	f := float32(current)

	err := wb.setCurrent(f)
	if err == nil {
		wb.current = f
	}

	return err
}

var _ api.ChargeTimer = (*Etrel)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Etrel) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegChargeTime, 4)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint64(b)) * time.Second, nil
}

var _ api.Meter = (*Etrel)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Etrel) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32(b) * 1e3), err
}

// var _ api.MeterEnergy = (*Etrel)(nil)

// // TotalEnergy implements the api.MeterEnergy interface
// func (wb *Etrel) TotalEnergy() (float64, error) {
// 	b, err := wb.conn.ReadInputRegisters(etrelRegTotalEnergy, 2)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return float64(binary.BigEndian.Uint32(b)) / 10, err
// }

var _ api.ChargeRater = (*Etrel)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Etrel) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegSessionEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32(b)), err
}

var _ api.MeterCurrent = (*Etrel)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Etrel) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range etrelRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, float64(encoding.Float32(b)))
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Diagnosis = (*Etrel)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Etrel) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(etrelRegModel, 10); err == nil {
		fmt.Printf("Model:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(etrelRegSerial, 10); err == nil {
		fmt.Printf("Serial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(etrelRegHWVersion, 5); err == nil {
		fmt.Printf("Hardware:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(etrelRegSWVersion, 5); err == nil {
		fmt.Printf("Software:\t%s\n", b)
	}
}
