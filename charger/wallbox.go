package charger

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/wallbox"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
)

// https://github.com/cliviu74/wallbox

// Wallbox charger implementation
type Wallbox struct {
	*request.Helper
	id      int
	current int64
}

func init() {
	registry.Add("wallbox", NewWallboxFromConfig)
	registry.Add("pulsar", NewWallboxFromConfig)
}

// NewWallboxFromConfig creates a Wallbox charger from generic config
func NewWallboxFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User     string
		Password string
		ID       int
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewWallbox(cc.User, cc.Password, cc.ID)
}

// NewWallbox creates Wallbox charger
func NewWallbox(user, password string, id int) (*Wallbox, error) {
	log := util.NewLogger("wallbox")

	c := &Wallbox{
		Helper:  request.NewHelper(log),
		id:      id,
		current: 6,
	}

	uri := fmt.Sprintf("%s/auth/token/user", wallbox.ApiURI)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        request.JSONContent,
		"Authorization": transport.BasicAuthHeader(user, password),
	})

	var res wallbox.Token
	if err == nil {
		err = c.DoJSON(req, &res)
	}

	if err == nil {
		c.Client.Transport = &transport.Decorator{
			Base: c.Client.Transport,
			Decorator: transport.DecorateHeaders(map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", res.Jwt),
			}),
		}
	}

	if id == 0 {
		var groups wallbox.Groups
		if err == nil {
			uri = fmt.Sprintf("%s/v3/chargers/groups", wallbox.ApiURI)
			err = c.GetJSON(uri, &groups)
		}

		chargers := lo.Flatten(lo.Map(groups.Result.Groups, func(g wallbox.Group, _ int) []int {
			return lo.Map(g.Chargers, func(c wallbox.Charger, _ int) int {
				return c.ID
			})
		}))

		if len(chargers) == 1 {
			c.id = chargers[0]
		} else {
			err = fmt.Errorf("found chargers: %v", chargers)
		}
	}

	return c, err
}

// Status implements the api.Charger interface
func (c *Wallbox) Status() (api.ChargeStatus, error) {
	var res wallbox.ChargerStatus

	uri := fmt.Sprintf("%s/chargers/status/%d", wallbox.ApiURI, c.id)
	err := c.GetJSON(uri, &res)

	if err != nil && res.Msg != "" {
		err = fmt.Errorf("%s: %w", res.Msg, err)
	}

	status := api.StatusA

	switch res.Status() {
	case wallbox.WAITING, wallbox.PAUSED:
		status = api.StatusB
	case wallbox.CHARGING:
		status = api.StatusC
	case wallbox.ERROR:
		status = api.StatusF
	}

	return status, err
}

// Enabled implements the api.Charger interface
func (c *Wallbox) Enabled() (bool, error) {
	var res wallbox.ChargerStatus

	uri := fmt.Sprintf("%s/chargers/status/%d", wallbox.ApiURI, c.id)
	err := c.GetJSON(uri, &res)

	if err != nil && res.Msg != "" {
		err = fmt.Errorf("%s: %w", res.Msg, err)
	}

	return res.ConfigData.MaxChargingCurrent > 0, err
}

// Enable implements the api.Charger interface
func (c *Wallbox) Enable(enable bool) error {
	var curr int64
	if enable {
		curr = c.current
	}
	return c.setCurrent(curr)
}

func (c *Wallbox) setCurrent(current int64) error {
	data := fmt.Sprintf(`{ "maxChargingCurrent":%d }`, current)

	uri := fmt.Sprintf("%s/v2/charger/%d", wallbox.ApiURI, c.id)
	req, err := request.New(http.MethodPut, uri, strings.NewReader(data), request.JSONEncoding)
	if err == nil {
		var res wallbox.Error
		if err = c.DoJSON(req, &res); err != nil && res.Msg != "" {
			err = fmt.Errorf("%s: %w", res.Msg, err)
		}
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Wallbox) MaxCurrent(current int64) error {
	err := c.setCurrent(current)
	if err == nil {
		c.current = current
	}
	return err
}
