// nmon2influxdb
// import nmon data in InfluxDB
// author: adejoux@djouxtech.net

package nmon

import (
	"errors"
	"fmt"
	"log"
	"math"
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
	FCs         map[string]FCstruct
	DF			map[string]DFstruct
}
type FCstruct struct {
	wwpn        string
	speed       string
	att         string
	lipcnt      string
	noscnt      string
	errframe    string
	dumpframe   string
	linkfail    string
	losssync    string
	losssig     string
	invtx       string
	invcrc      string
}
type DFstruct struct {
	mount   	string
	blocks_mb	float64
	used_mb		float64
	used_pct	float64
	iused		float64
	iused_pct	float64
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
	return &Nmon{DataSeries: make(map[string]DataSerie), TimeStamps: make(map[string]string), FCs: make(map[string]FCstruct), DF: make(map[string]DFstruct)}

}

// BuildPoint create a point and convert string value to float when possible  VG - couldn't find any matches/calls for this function !!! TODO! 
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

// GetColumnInDataserie returns column number for the serie
func (nmon *Nmon) GetColumnInDataserie(serie string, column string) (Number int, err error) {
	// VG
	d, ok := nmon.DataSeries[serie]
	if !ok {
		errorMessage := fmt.Sprint("Serie %s does not exists", serie)
		err = errors.New(errorMessage)
		return
	}
	for n, value := range d.Columns {
		if value == column {
			Number = n + 2
			return 
		}
	}
	errorMessage := fmt.Sprint("Column %s not fount in Serie %s", column, serie)
	err = errors.New(errorMessage)
	return
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

//InitNmonTemplate init nmon structure when creating dashboard  VG - calls from stats and dashboard !!
func InitNmonTemplate(config *nmon2influxdblib.Config) (nmon *Nmon) {
	nmon = NewNmon()
	nmon.Config = config
	if config.Debug {
		log.Printf("configuration: %+v\n", config.Sanitized())
	}

	nmon.SetLocation(config.Timezone)
	return
}

//InitNmon init nmon structure for nmon file import  - VG - MAIN function in nmon INITIALIZATION !!!
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

	//VG build Series struct for ALL Measurements, but filtar during the import.
	//var userSkipRegexp *regexp.Regexp
	//
	//if len(config.ImportSkipMetrics) > 0 {
	//	skipped := strings.Replace(config.ImportSkipMetrics, ",", "|", -1)
	//	userSkipRegexp = regexp.MustCompile(skipped)
	//}
	badtext := fmt.Sprintf("%s%s",nmonFile.Delimiter,nmonFile.Delimiter)
        var badRegexp = regexp.MustCompile(badtext)
	for _, line := range lines {

		if cpuallRegexp.MatchString(line) && !config.ImportAllCpus {	// var cpuallRegexp = regexp.MustCompile(`^CPU\d+|^SCPU\d+|^PCPU\d+`)
			continue													// CPU162,CPU 162 wscln2,User%,Sys%,Wait%,Idle%
		}																// CPU162,T0001,0.0,0.0,0.0,100.0

		if diskallRegexp.MatchString(line) && config.ImportSkipDisks {	// var diskallRegexp = regexp.MustCompile(`^DISK`)
			continue
		}

		if timeRegexp.MatchString(line) {								// var timeRegexp = regexp.MustCompile(`^ZZZZ.(T\d+).(.*)$`)
			matched := timeRegexp.FindStringSubmatch(line)				// ZZZZ,T0001,18:30:17,06-NOV-2022
			nmon.TimeStamps[matched[1]] = matched[2]					// nmon.Timestamps["T0001"] = "18:30:17,06-NOV-2022"
			continue
		}

		if hostRegexp.MatchString(line) {								// var hostRegexp = regexp.MustCompile(`^AAA.host.(\S+)`)
			matched := hostRegexp.FindStringSubmatch(line)				// AAA,host,wscln2   - only 1 match
			nmon.Hostname = strings.ToLower(matched[1])
			continue
		}

		if serialRegexp.MatchString(line) {								// var serialRegexp = regexp.MustCompile(`^AAA.SerialNumber.(\S+)`)
			matched := serialRegexp.FindStringSubmatch(line)			// ^AAA.SerialNumber.(\S+)   - only 1 match
			nmon.Serial = strings.ToUpper(matched[1])
			continue
		}

		//if osRegexp.MatchString(line) {
		//	matched := osRegexp.FindStringSubmatch(line)
		//	nmon.OS = strings.ToLower(matched[1])
		//	continue
		//}

		// VG ++

		if viosverRegexp.MatchString(line) {							// var viosverRegexp = regexp.MustCompile(`^AAA.VIOS.(\S+)`)
            matched := viosverRegexp.FindStringSubmatch(line)
            nmon.OS = "vios"
			nmon.OStl = nmon.OSver
            nmon.OSver = strings.ToLower(matched[1])
            continue
        }
		if aixverRegexp.MatchString(line) {								// var aixverRegexp = regexp.MustCompile(`^AAA.AIX.(\S+)`)
			matched := aixverRegexp.FindStringSubmatch(line)
			if nmon.OS == "vios" {
				nmon.OStl = strings.ToLower(matched[1])
				continue
			}
			nmon.OS = "aix"
			nmon.OSver = strings.ToLower(matched[1])					// 
			continue
		}
		if aixtlRegexp.MatchString(line) {								// var aixtlRegexp = regexp.MustCompile(`^AAA.TL.(\d+)`)
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
		if aixfcwwpnRegexp.MatchString(line) {
			matched := aixfcwwpnRegexp.FindStringSubmatch(line)
			fcname := matched[1]
			fcstruct := nmon.FCs[fcname]
			fcstruct.wwpn = matched[2]
			nmon.FCs[fcname] = fcstruct
			continue
		}
		if aixfcportspeedRegexp.MatchString(line) {
            matched := aixfcportspeedRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.speed = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfcattentionRegexp.MatchString(line) {
            matched := aixfcattentionRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.att = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfclipcountRegexp.MatchString(line) {
            matched := aixfclipcountRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.lipcnt = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfcnoscountRegexp.MatchString(line) {
            matched := aixfcnoscountRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.noscnt = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfcerrframesRegexp.MatchString(line) {
            matched := aixfcerrframesRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.errframe = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfcdumpedframesRegexp.MatchString(line) {
            matched := aixfcdumpedframesRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.dumpframe = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfclinkfailureRegexp.MatchString(line) {
            matched := aixfclinkfailureRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.linkfail = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfclossofsyncRegexp.MatchString(line) {
            matched := aixfclossofsyncRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.losssync = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfclossofsignalRegexp.MatchString(line) {
            matched := aixfclossofsignalRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.losssig = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfcinvalidtxRegexp.MatchString(line) {
            matched := aixfcinvalidtxRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.invtx = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if aixfcinvalidcrcRegexp.MatchString(line) {
            matched := aixfcinvalidcrcRegexp.FindStringSubmatch(line)
            fcname := matched[1]
            fcstruct := nmon.FCs[fcname]
            fcstruct.invcrc = matched[2]
            nmon.FCs[fcname] = fcstruct
            continue
        }
		if dfRegexp.MatchString(line) {
			matched := dfRegexp.FindStringSubmatch(line)
			elems   := strings.Fields(matched[1])
			fvalues := []float64{}

			if nmon.OS == "linux" {
				// DFfieldsLinux := []{"filesystem", "blocks_mb", "used_mb", "available_mb", "used%", "mount to"}
				if (len(elems) < 6) || (elems[0] == "Filesystem") || (elems[0] == "shm") || (elems[0] == "overlay") {
					continue
				}				
				for _, rawvalue := range elems[1:5] {
					value, parseErr := strconv.ParseFloat(strings.Replace(rawvalue, "%", "", 1), 64)
					if parseErr != nil || math.IsNaN(value) {
						continue
					}
					fvalues = append(fvalues, value)
				}
				if len(fvalues) < 4 {
					continue
				}
				if elems[0][0:4] == "ddev" {
					elems[0] = "/" + elems[0][1:]
				}
				fields, ok := nmon.DF[elems[0]]
				if !ok || (elems[5] == "/run") {
					fields.mount      = elems[5]
					fields.blocks_mb  = fvalues[0]
					fields.used_mb    = fvalues[1]
					fields.used_pct   = fvalues[1]/fvalues[0]*100
					fields.iused      = 0
					fields.iused_pct  = 0
					nmon.DF[elems[0]] = fields
				}
			} else {
				// DFfieldsAIX   := []{"filesystem", "blocks_mb", "free_mb", "used%", "iused", "iused%", "mount to"}
				if (len(elems) < 7) || (elems[0] == "Filesystem") || (elems[0] == "dproc") {
					continue
				}				
				for _, rawvalue := range elems[1:6] {
					value, parseErr := strconv.ParseFloat(strings.Replace(rawvalue, "%", "", 1), 64)
					if parseErr != nil || math.IsNaN(value) {
						// fmt.Printf("Conv err. rawval=%s, value=%v, index=%d, elems=%v\n", rawvalue, value, len(fvalues), elems)
						continue
					}
					fvalues = append(fvalues, value)
				}
				if len(fvalues) < 5 {
					continue
				}
				if elems[0][0:4] == "ddev" {
					elems[0] = "/" + elems[0][1:]
				}
				fields := nmon.DF[elems[0]]
				fields.mount      = elems[6]
				fields.blocks_mb  = fvalues[0]
				fields.used_mb    = fvalues[0] - fvalues[1]
				fields.used_pct   = fvalues[2]
				fields.iused      = fvalues[3]
				fields.iused_pct   = fvalues[4]
				nmon.DF[elems[0]] = fields
			}
			continue
		}
		//VG --

		if infoRegexp.MatchString(line) {								// var infoRegexp = regexp.MustCompile(`^AAA.(.*)`)
			matched := infoRegexp.FindStringSubmatch(line)
			nmon.AppendText(matched[1])
			continue
		}

		if !headerRegexp.MatchString(line) {							// var headerRegexp = regexp.MustCompile(`^AAA|^BBB|^UARG|\WT\d{4,16}`)
			if len(line) == 0 {											//  Records that describe series/
				continue												//  Like...
			}															//    CPU_ALL,CPU Total wscln2,User%,Sys%,Wait%,Idle%,Busy,PhysicalCPUs 
																		//    CPU01,CPU 1 wscln2,User%,Sys%,Wait%,Idle% 
																		//    MEM,Memory wscln2,Real Free %,Virtual free %,Real free(MB),Virtual free(MB),Real total(MB),Virtual total(MB)
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
			name := elems[0]								// Name of serie, like CPU_ALL, MEM, etc
			//VG build Series struct for ALL Measurements, but filtar during the import.
			//if len(config.ImportSkipMetrics) > 0 {			//
			//	if userSkipRegexp.MatchString(name) {		//  Skip series that are mentioned in config userSkipRegexp 
			//		continue
			//	}
			//}

			if config.Debug == true {
				log.Printf("Adding serie %s\n", name)
			}

			dataserie := nmon.DataSeries[name]				// make new dataserie -> array/slice of strings
			dataserie.Columns = elems[2:]					//   arrays of strings - colums name
			nmon.DataSeries[name] = dataserie				// map - [data-serie-name] -> array of colum names
		}
	}		// for _, line := range lines .....

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
		for dev, devval := range nmon.FCs {
			log.Printf("FC statisticts report:\n" +
			"%s.wwpn        = %s\n" +
                        "%s.speed       = %s\n" +
                        "%s.att         = %s\n" +
                        "%s.lipcnt      = %s\n" +
                        "%s.noscnt      = %s\n" +
                        "%s.errframe    = %s\n" +
                        "%s.dumpframe   = %s\n" +
                        "%s.linkfail    = %s\n" +
                        "%s.losssync    = %s\n" +
                        "%s.losssig     = %s\n" +
                        "%s.invtx       = %s\n" +
                        "%s.invcrc      = %s\n",
                        dev, devval.wwpn,
                        dev, devval.speed,
                        dev, devval.att,
                        dev, devval.lipcnt,
                        dev, devval.noscnt,
                        dev, devval.errframe,
                        dev, devval.dumpframe,
                        dev, devval.linkfail,
                        dev, devval.losssync,
                        dev, devval.losssig,
                        dev, devval.invtx,
                        dev, devval.invcrc)
		}
	}		// if config.Debug
	//VG--
	return
}			// func InitNmon(...) (nmon *Nmon)

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
