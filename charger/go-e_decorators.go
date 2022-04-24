package charger

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateGoE(base *GoE, meterEnergy func() (float64, error), chargePhases func(phases int) error, alarmClock func() error) api.Charger {
	switch {
	case alarmClock == nil && chargePhases == nil && meterEnergy == nil:
		return base

	case alarmClock == nil && chargePhases == nil && meterEnergy != nil:
		return &struct {
			*GoE
			api.MeterEnergy
		}{
			GoE: base,
			MeterEnergy: &decorateGoEMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case alarmClock == nil && chargePhases != nil && meterEnergy == nil:
		return &struct {
			*GoE
			api.ChargePhases
		}{
			GoE: base,
			ChargePhases: &decorateGoEChargePhasesImpl{
				chargePhases: chargePhases,
			},
		}

	case alarmClock == nil && chargePhases != nil && meterEnergy != nil:
		return &struct {
			*GoE
			api.ChargePhases
			api.MeterEnergy
		}{
			GoE: base,
			ChargePhases: &decorateGoEChargePhasesImpl{
				chargePhases: chargePhases,
			},
			MeterEnergy: &decorateGoEMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case alarmClock != nil && chargePhases == nil && meterEnergy == nil:
		return &struct {
			*GoE
			api.AlarmClock
		}{
			GoE: base,
			AlarmClock: &decorateGoEAlarmClockImpl{
				alarmClock: alarmClock,
			},
		}

	case alarmClock != nil && chargePhases == nil && meterEnergy != nil:
		return &struct {
			*GoE
			api.AlarmClock
			api.MeterEnergy
		}{
			GoE: base,
			AlarmClock: &decorateGoEAlarmClockImpl{
				alarmClock: alarmClock,
			},
			MeterEnergy: &decorateGoEMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case alarmClock != nil && chargePhases != nil && meterEnergy == nil:
		return &struct {
			*GoE
			api.AlarmClock
			api.ChargePhases
		}{
			GoE: base,
			AlarmClock: &decorateGoEAlarmClockImpl{
				alarmClock: alarmClock,
			},
			ChargePhases: &decorateGoEChargePhasesImpl{
				chargePhases: chargePhases,
			},
		}

	case alarmClock != nil && chargePhases != nil && meterEnergy != nil:
		return &struct {
			*GoE
			api.AlarmClock
			api.ChargePhases
			api.MeterEnergy
		}{
			GoE: base,
			AlarmClock: &decorateGoEAlarmClockImpl{
				alarmClock: alarmClock,
			},
			ChargePhases: &decorateGoEChargePhasesImpl{
				chargePhases: chargePhases,
			},
			MeterEnergy: &decorateGoEMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type decorateGoEAlarmClockImpl struct {
	alarmClock func() (error)
}

func (impl *decorateGoEAlarmClockImpl) WakeUp() error {
	return impl.alarmClock()
}

type decorateGoEChargePhasesImpl struct {
	chargePhases func(int) error
}

func (impl *decorateGoEChargePhasesImpl) Phases1p3p(phases int) error {
	return impl.chargePhases(phases)
}

type decorateGoEMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateGoEMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
