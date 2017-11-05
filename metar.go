package avwx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	baseURL = "https://avwx.rest/api/metar/"
	options = "?options=info"
)

var conditions = map[string]string{
	"RA":   "RAIN",
	"DZ":   "DRIZZLE",
	"SN":   "SNOW",
	"SG":   "SNOW GRAINS",
	"IC":   "ICE CRYSTALS",
	"PL":   "ICE PELLETS",
	"GR":   "HAIL",
	"GS":   "SMALL HAIL/SNOW PELLETS",
	"UP":   "UNKNOWN PRECIPITATION",
	"BR":   "MIST",
	"FG":   "FOG",
	"FU":   "SMOKE",
	"VA":   "VOLCANIC ASH",
	"SA":   "SAND",
	"HZ":   "HAZE",
	"PY":   "SPRAY",
	"DU":   "DUST",
	"SQ":   "SQUALL",
	"SS":   "SANDSTORM",
	"DS":   "DUSTSTORM",
	"PO":   "WELL DEVELOPED DUST/SAND WHIRLS",
	"FC":   "FUNNEL CLOUD",
	"VC":   "IN VICINITY",
	"MI":   "SHALLOW",
	"BC":   "PATCHES",
	"SH":   "SHOWERS",
	"PR":   "PARTIAL",
	"TS":   "THUNDERSTORM",
	"TSRA": "THUNDERSTORM/HEAVY RAIN",
	"BL":   "BLOWING",
	"DR":   "DRIFTING",
	"FZ":   "FREEZING",
}

var coverage = map[string]string{
	"FEW": "FEW",
	"SKC": "SKY CLEAR",
	"OVC": "OVERCAST",
	"SCT": "SCATTERED",
	"BKN": "BROKEN",
	"VV":  "VERTICLE VISIBILITY",
}

var cloudTypes = map[string]string{
	"CB":    "CUMULONIMBUS",
	"TCU":   "TOWERING CUMULUS",
	"CBMAM": "CUMULONIMBUS MAMMATUS",
}

// FetchMetar fetches the current METAR for given station represented by a valid ICAO airport code.
func FetchMetar(station string) *MetarResponse {
	//start := time.Now()
	url := baseURL + station + options

	metarResp := new(MetarResponse)
	metarResp.ICAO = station

	resp, err := http.Get(url)
	if err != nil {
		metarResp.Error = err
		return metarResp
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metarResp.Error = fmt.Errorf("Query failed: %s", resp.Status)
		return metarResp
	}

	var metar Metar
	if err := json.NewDecoder(resp.Body).Decode(&metar); err != nil {
		metarResp.Error = err
		return metarResp
	}
	decodeMetar(&metar)
	metarResp.Metar = metar
	//fmt.Printf("\nFetched: %s in %.2fs\n", station, time.Since(start).Seconds())
	return metarResp
}

func decodeMetar(metar *Metar) {

	altimeter, _ := strconv.ParseFloat(metar.Altimeter, 64)
	metar.Altimeter = strconv.FormatFloat(altimeter/100, 'f', 2, 64)

	metar.Temperature = strings.Replace(metar.Temperature, "M", "-", 1)
	temp, _ := strconv.ParseFloat(metar.Temperature, 64)
	metar.TemperatureF = fmt.Sprintf("%.1f", cToF(temp))
	metar.Temperature = fmt.Sprintf("%.1f", temp)

	metar.Dewpoint = strings.Replace(metar.Dewpoint, "M", "-", 1)
	dewpoint, _ := strconv.ParseFloat(metar.Dewpoint, 64)
	metar.DewpointF = fmt.Sprintf("%.1f", cToF(dewpoint))
	metar.Dewpoint = fmt.Sprintf("%.1f", dewpoint)

	windDegrees, _ := strconv.ParseInt(metar.WindDirection, 10, 32)
	metar.WindDirectionDesc = GetDirectionDesc(windDegrees)

	for _, condition := range metar.Conditions {
		modifier := ""
		vicinity := false

		if strings.HasPrefix(condition, "VC") {
			vicinity = true
			condition = condition[2:]
		}
		if strings.HasPrefix(condition, "-") {
			modifier = "LIGHT"
			condition = condition[1:]
		} else if strings.HasPrefix(condition, "+") {
			modifier = "HEAVY"
			condition = condition[1:]
		}

		conditionDec := new(ConditionDec)
		conditionDec.Desc = conditions[condition]
		conditionDec.Modifier = modifier
		if vicinity {
			conditionDec.Other = "IN VICINITY"
		}
		metar.ConditionsDec = append(metar.ConditionsDec, *conditionDec)
	}

	for _, layer := range metar.CloudLayers {
		cloudLayerDec := new(CloudLayerDec)
		cloudLayerDec.Coverage = coverage[layer[0]]
		height, _ := strconv.ParseInt(layer[1], 10, 64)
		cloudLayerDec.HeightFt = fmt.Sprintf("%d", height*100)
		if len(layer) > 2 {
			cloudLayerDec.Type = cloudTypes[layer[2]]
		}
		metar.CloudLayersDec = append(metar.CloudLayersDec, *cloudLayerDec)
	}
}

func GetDirectionDesc(degrees int64) string {
	switch {
	case (degrees > 349 && degrees <= 360) || (degrees >= 0 && degrees <= 11):
		return "N"
	case degrees > 11 && degrees <= 34:
		return "NNE"
	case degrees > 34 && degrees <= 56:
		return "NE"
	case degrees > 56 && degrees <= 79:
		return "ENE"
	case degrees > 79 && degrees <= 101:
		return "E"
	case degrees > 101 && degrees <= 124:
		return "ESE"
	case degrees > 124 && degrees <= 146:
		return "SE"
	case degrees > 146 && degrees <= 169:
		return "SSE"
	case degrees > 169 && degrees <= 191:
		return "S"
	case degrees > 191 && degrees <= 214:
		return "SSW"
	case degrees > 214 && degrees <= 236:
		return "SW"
	case degrees > 236 && degrees <= 259:
		return "WSW"
	case degrees > 259 && degrees <= 281:
		return "W"
	case degrees > 281 && degrees <= 304:
		return "WNW"
	case degrees > 304 && degrees <= 326:
		return "NW"
	case degrees > 326 && degrees <= 349:
		return "NNW"
	default:
		return ""
	}
}

func FormatICAO(icao string) (string, error) {
	len := len(icao)

	if len < 3 || len > 4 {
		return icao, fmt.Errorf("Invalid airport code: %s", icao)
	}

	icao = strings.ToUpper(icao)
	if len < 4 {
		icao = "K" + icao
	}

	return icao, nil
}

func cToF(c float64) float64 {
	return c*9/5 + 32
}

type Metar struct {
	Altimeter         string
	Dewpoint          string
	DewpointF         string
	FlightRules       string `json:"Flight-Rules"`
	RawReport         string `json:"Raw-Report"`
	Remarks           string
	Station           string
	Temperature       string
	TemperatureF      string
	Time              string
	Visibility        string
	WindDirection     string `json:"Wind-Direction"`
	WindDirectionDesc string
	WindGust          string     `json:"Wind-Gust"`
	WindSpeed         string     `json:"Wind-Speed"`
	CloudLayers       [][]string `json:"Cloud-List"`
	CloudLayersDec    []CloudLayerDec
	Conditions        []string `json:"Other-List"`
	ConditionsDec     []ConditionDec
	Error             string
	LocationInfo      LocationInfo `json:"Info"`
}

type LocationInfo struct {
	City    string
	Country string
	Name    string
	State   string
}

type ConditionDec struct {
	Modifier string
	Desc     string
	Other    string
}

type CloudLayerDec struct {
	Coverage string
	HeightFt string
	Type     string
}

type MetarResponse struct {
	Metar Metar
	Error error
	ICAO  string
}
