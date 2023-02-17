// nmon2influxdb
// import nmon data in InfluxDB
// author: adejoux@djouxtech.net

package nmon

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/adejoux/nmon2influxdb/nmon2influxdblib"
)

// Nmon structure used to manage nmon files
type Nmon struct {
	Hostname    string
	Serial      string
	OS          string
	OSver       string
	OStl        string
	MT          string
	CPUs        string
	SMT         string
	CPUtype     string
	CPUmode     string
	FW          string
	uptime      string
	LPARnr	    string
	LPARname    string
	TimeStamps  map[string]string
	TextContent string
	DataSeries  map[string]DataSerie
	Debug       bool
	Config      *nmon2influxdblib.Config
	starttime   time.Time
	stoptime    time.Time
	Location    *time.Location
	TagParsers  nmon2influxdblib.TagParsers
}

// DataSerie structure contains the columns and points to insert in InfluxDB
type DataSerie struct {
	Columns []string
}

// AppendText add text section to dashboard
func (nmon *Nmon) AppendText(text string) {
	nmon.TextContent += nmon2influxdblib.ReplaceComma(text)
}

// NewNmon initialize a Nmon structure
func NewNmon() *Nmon {
	return &Nmon{DataSeries: make(map[string]DataSerie), TimeStamps: make(map[string]string)}

}

// BuildPoint create a point and convert string value to float when possible
func (nmon *Nmon) BuildPoint(serie string, values []string) map[string]interface{} {
	columns := nmon.DataSeries[serie].Columns
	//TODO check output
	point := make(map[string]interface{})

	for i, rawvalue := range values {
		// try to convert string to integer
		value, err := strconv.ParseFloat(rawvalue, 64)
		if err != nil {
			//if not working, use string
			point[columns[i]] = rawvalue
		} else {
			//send integer if it worked
			point[columns[i]] = value
		}
	}

	return point
}

//GetTimeStamp retrieves the TimeStamp corresponding to the entry
func (nmon *Nmon) GetTimeStamp(label string) (timeStamp string, err error) {
	if t, ok := nmon.TimeStamps[label]; ok {
		timeStamp = t
	} else {
		errorMessage := fmt.Sprintf("TimeStamp %s not found", label)
		err = errors.New(errorMessage)
	}
	return
}

//InitNmonTemplate init nmon structure when creating dashboard
func InitNmonTemplate(config *nmon2influxdblib.Config) (nmon *Nmon) {
	nmon = NewNmon()
	nmon.Config = config
	if config.Debug {
		log.Printf("configuration: %+v\n", config.Sanitized())
	}

	nmon.SetLocation(config.Timezone)
	return
}

//InitNmon init nmon structure for nmon file import
func InitNmon(config *nmon2influxdblib.Config, nmonFile nmon2influxdblib.File) (nmon *Nmon) {
	//Xcps  := 0
	//Xsockets := 0
	AIXcpus := "0"
	Linuxcpus := "0"
	nmon = NewNmon()
	nmon.Config = config
	nmon.CPUmode = ""
	if config.Debug {
		log.Printf("configuration: %+v\n", config.Sanitized())
	}

	nmon.SetLocation(config.Timezone)
	nmon.Debug = config.Debug

	lines := nmonFile.Content()

	var userSkipRegexp *regexp.Regexp

	if len(config.ImportSkipMetrics) > 0 {
		skipped := strings.Replace(config.ImportSkipMetrics, ",", "|", -1)
		userSkipRegexp = regexp.MustCompile(skipped)
	}
	badtext := fmt.Sprintf("%s%s",nmonFile.Delimiter,nmonFile.Delimiter)
        var badRegexp = regexp.MustCompile(badtext)
	for _, line := range lines {

		if cpuallRegexp.MatchString(line) && !config.ImportAllCpus {
			continue
		}

		if diskallRegexp.MatchString(line) && config.ImportSkipDisks {
			continue
		}

		if timeRegexp.MatchString(line) {
			matched := timeRegexp.FindStringSubmatch(line)
			nmon.TimeStamps[matched[1]] = matched[2]
			continue
		}

		if hostRegexp.MatchString(line) {
			matched := hostRegexp.FindStringSubmatch(line)
			nmon.Hostname = strings.ToLower(matched[1])
			continue
		}

		if serialRegexp.MatchString(line) {
			matched := serialRegexp.FindStringSubmatch(line)
			nmon.Serial = strings.ToUpper(matched[1])
			continue
		}

		//if osRegexp.MatchString(line) {
		//	matched := osRegexp.FindStringSubmatch(line)
		//	nmon.OS = strings.ToLower(matched[1])
		//	continue
		//}

		// VG ++

		if viosverRegexp.MatchString(line) {
                        matched := viosverRegexp.FindStringSubmatch(line)
                        nmon.OS = "vios"
			nmon.OStl = nmon.OSver
                        nmon.OSver = strings.ToLower(matched[1])
                        continue
                }
		if aixverRegexp.MatchString(line) {
			matched := aixverRegexp.FindStringSubmatch(line)
			if nmon.OS == "vios" {
				nmon.OStl = strings.ToLower(matched[1])
				continue
			}
			nmon.OS = "aix"
			nmon.OSver = strings.ToLower(matched[1])
			continue
		}
		if aixtlRegexp.MatchString(line) {
			if nmon.OS == "vios" {
                                continue
                        }
                        matched := aixtlRegexp.FindStringSubmatch(line)
                        nmon.OStl = matched[1]
                        continue
                }

		if lparnumbernameRegexp.MatchString(line) {
                        matched := lparnumbernameRegexp.FindStringSubmatch(line)
                        nmon.LPARnr = matched[1]
			nmon.LPARname = matched[2]
                        continue
                }

		if aixmtRegexp.MatchString(line) {
                        matched := aixmtRegexp.FindStringSubmatch(line)
                        //nmon.MT = strings.ToLower(matched[1])
			nmon.MT = strings.ToUpper(matched[1])
                        continue
                }
		if aixcpusRegexp.MatchString(line) {
			matched := aixcpusRegexp.FindStringSubmatch(line)
			AIXcpus = matched[1]
			//log.Printf("aixcpus matched. line = %s. AIXcpus = %s\n", line, AIXcpus)
			continue
		}
		if aixsmtRegexp.MatchString(line) {
                        matched := aixsmtRegexp.FindStringSubmatch(line)
                        nmon.SMT = matched[1]
                        continue
		}
		if aixcputypeRegexp.MatchString(line) {
			matched := aixcputypeRegexp.FindStringSubmatch(line)
			regex := regexp.MustCompile(`(?i)PowerPC_`)
			str := regex.ReplaceAllString(matched[1], "")
			nmon.CPUtype = str
			continue
		}
		if aixcpumodeRegexp.MatchString(line) {
			matched := aixcpumodeRegexp.FindStringSubmatch(line)
			regex := regexp.MustCompile(`\s+`)
			str := regex.ReplaceAllString(matched[1], "")
			nmon.CPUmode = str
			continue
		}
		if aixfirmwareRegexp.MatchString(line) {
			matched := aixfirmwareRegexp.FindStringSubmatch(line)
			nmon.FW = matched[2]
			continue
		}

		if linuxserialRegexp.MatchString(line) {
                        matched := linuxserialRegexp.FindStringSubmatch(line)
                        nmon.Serial = strings.ToUpper(matched[1])
                        continue
                }

		if linuxverRegexp.MatchString(line) {
                        matched := linuxverRegexp.FindStringSubmatch(line)
                        //nmon.OSver = strings.ToLower(matched[1])
                        nmon.OSver = matched[1]
                        continue
                }

		if linuxkernelRegexp.MatchString(line) {
                        matched := linuxkernelRegexp.FindStringSubmatch(line)
			nmon.OS = "linux"
                        nmon.OStl = matched[1]
                        continue
                }

		if linuxmtRegexp.MatchString(line) {
                        matched := linuxmtRegexp.FindStringSubmatch(line)
                        nmon.MT = strings.ToUpper(matched[1])
                        continue
                }

		//if linuxcpsRegexp.MatchString(line) {
                //        matched := linuxcpsRegexp.FindStringSubmatch(line)
		//	// try to convert string to integer
                //        converted, parseErr := strconv.Atoi(matched[1])
                //        if parseErr != nil {
		//		continue
                //        }
		//	Xcps = converted
                //        continue
                //}

		//if linuxsocketsRegexp.MatchString(line) {
                //        matched := linuxsocketsRegexp.FindStringSubmatch(line)
                //        // try to convert string to integer
                //        converted, parseErr := strconv.Atoi(matched[1])
                //        if parseErr != nil {
                //                continue
                //        }
		//	Xsockets = converted
                //        continue
                //}
		if linuxcpusRegexp.MatchString(line) {
                        matched := linuxcpusRegexp.FindStringSubmatch(line)
                        Linuxcpus = matched[1]
			//log.Printf("linuxcpus matched. line = %s. Linuxcpus = %s\n", line, Linuxcpus)
                        continue
                }
		if x86cpusRegexp.MatchString(line) {
                        matched := x86cpusRegexp.FindStringSubmatch(line)
                        Linuxcpus = matched[1]
                        //log.Printf("linuxcpus matched. line = %s. Linuxcpus = %s\n", line, Linuxcpus)
                        continue
                }
		if x86cpumodeRegexp.MatchString(line) {
                        matched := x86cpumodeRegexp.FindStringSubmatch(line)
                        nmon.CPUmode = matched[1]
                        continue
                }

		if linuxsmtRegexp.MatchString(line) {
                        matched := linuxsmtRegexp.FindStringSubmatch(line)
                        nmon.SMT = matched[1]
                        continue
                }

		if linuxcputypeRegexp.MatchString(line) {
                        matched := linuxcputypeRegexp.FindStringSubmatch(line)
                        nmon.CPUtype = matched[1]
			//nmon.CPUmode = ""
                        continue
                }

		if linuxfirmwareRegexp.MatchString(line) {
                        matched := linuxfirmwareRegexp.FindStringSubmatch(line)
                        nmon.FW = matched[1]
                        continue
                }
		if linuxbmcfirmwareRegexp.MatchString(line) {
                        matched := linuxbmcfirmwareRegexp.FindStringSubmatch(line)
                        nmon.FW = "bmcFW" + matched[1]
                        continue
                }
		if uptimeRegexp.MatchString(line) {
			matched := uptimeRegexp.FindStringSubmatch(line)
                        nmon.uptime = matched[1]
                        continue
                }

		//VG --

		if infoRegexp.MatchString(line) {
			matched := infoRegexp.FindStringSubmatch(line)
			nmon.AppendText(matched[1])
			continue
		}

		if !headerRegexp.MatchString(line) {
			if len(line) == 0 {
				continue
			}

			if badRegexp.MatchString(line) {
				continue
			}

			elems := strings.Split(line, nmonFile.Delimiter)

			if len(elems) < 3 {
				if config.Debug == true {
					log.Printf("ERROR: parsing the following line : %s\n", line)
				}
				continue
			}
			name := elems[0]
			if len(config.ImportSkipMetrics) > 0 {
				if userSkipRegexp.MatchString(name) {
					continue
				}
			}

			if config.Debug == true {
				log.Printf("Adding serie %s\n", name)
			}

			dataserie := nmon.DataSeries[name]
			dataserie.Columns = elems[2:]
			nmon.DataSeries[name] = dataserie
		}
	}
	//VG ++
	//if Xcps > 0 {
	//	nmon.CPUs = strconv.Itoa(Xcps * Xsockets)
	//}
	if AIXcpus == "0" {
		nmon.CPUs = Linuxcpus
	} else {
		nmon.CPUs = AIXcpus
	}
	if config.Debug {
		log.Printf("InitNmon results:\n" +
			"Hostname = %s\n" +
			"Serial   = %s\n" +
			"OS       = %s\n" +
			"OSver    = %s\n" +
			"OStl     = %s\n" +
			"MT       = %s\n" +
			"CPUs     = %s\n" +
			"SMT      = %s\n" +
			"CPUtype  = %s\n" +
			"CPUmode  = %s\n" +
			"FW       = %s\n" +
			"uptime   = %s\n" +
			"LPARnr   = %s\n" +
			"LPARname = %s\n",
			nmon.Hostname, nmon.Serial, nmon.OS, nmon.OSver, nmon.OStl, nmon.MT, nmon.CPUs, nmon.SMT, nmon.CPUtype, nmon.CPUmode, nmon.FW, nmon.uptime,
			nmon.LPARnr, nmon.LPARname)
	}


	//VG--
	return
}

//SetTimeFrame set the current timeframe for the dashboard
func (nmon *Nmon) SetTimeFrame() {
	keys := make([]string, 0, len(nmon.TimeStamps))
	for k := range nmon.TimeStamps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	nmon.starttime, _ = nmon.ConvertTimeStamp(nmon.TimeStamps[keys[0]])
	nmon.stoptime, _ = nmon.ConvertTimeStamp(nmon.TimeStamps[keys[len(keys)-1]])
}

// StartTime returns the starting timestamp for dashboard
func (nmon *Nmon) StartTime() string {
	if nmon.starttime == (time.Time{}) {
		nmon.SetTimeFrame()
	}
	return nmon.starttime.UTC().Format(time.RFC3339)
}

// StopTime returns the ending timestamp for dashboard
func (nmon *Nmon) StopTime() string {
	if nmon.stoptime == (time.Time{}) {
		nmon.SetTimeFrame()
	}
	return nmon.stoptime.UTC().Format(time.RFC3339)
}

const timeformat = "15:04:05 02-Jan-2006"

//SetLocation set the timezone used to input metrics in InfluxDB
func (nmon *Nmon) SetLocation(tz string) (err error) {
	var loc *time.Location
	if len(tz) > 0 {
		loc, err = time.LoadLocation(tz)
		if err != nil {
			loc = time.FixedZone("Europe/Paris", 2*60*60)
		}
	} else {
		timezone, _ := time.Now().In(time.Local).Zone()
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			loc = time.FixedZone("Europe/Paris", 2*60*60)
		}
	}

	nmon.Location = loc
	return
}

//ConvertTimeStamp convert the string timestamp in time.Time structure
func (nmon *Nmon) ConvertTimeStamp(s string) (time.Time, error) {
	var err error
	if s == "now" {
		return time.Now().Truncate(24 * time.Hour), err
	}

  //replace separator
  stamp := s[0:8] + " " + s[9:]
	t, err := time.ParseInLocation(timeformat, stamp, nmon.Location)
	return t, err
}

//DbURL generates InfluxDB server url
func (nmon *Nmon) DbURL() string {
	return "http://" + nmon.Config.InfluxdbServer + ":" + nmon.Config.InfluxdbPort
}
