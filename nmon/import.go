// nmon2influxdb
// import nmon report in InfluxDB
// author: adejoux@djouxtech.net

package nmon

import (
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/adejoux/influxdbclient"
	"github.com/adejoux/nmon2influxdb/nmon2influxdblib"
	"github.com/urfave/cli/v2"
)

var hostRegexp = regexp.MustCompile(`^AAA.host.(\S+)`)
var serialRegexp = regexp.MustCompile(`^AAA.SerialNumber.(\S+)`)
var osRegexp = regexp.MustCompile(`^AAA.*(Linux|AIX)`)
var timeRegexp = regexp.MustCompile(`^ZZZZ.(T\d+).(.*)$`)
var intervalRegexp = regexp.MustCompile(`^AAA.interval.(\d+)`)
var headerRegexp = regexp.MustCompile(`^AAA|^BBB|^UARG|\WT\d{4,16}`)
var infoRegexp = regexp.MustCompile(`^AAA.(.*)`)
var cpuallRegexp = regexp.MustCompile(`^CPU\d+|^SCPU\d+|^PCPU\d+`)
var diskallRegexp = regexp.MustCompile(`^DISK`)
var skipRegexp = regexp.MustCompile(`T0+\W|^Z|^TOP.%CPU`)
var statsRegexp = regexp.MustCompile(`\W(T\d{4,16})`)
var topRegexp = regexp.MustCompile(`^TOP.\d+.(T\d+)`)
var nfsRegexp = regexp.MustCompile(`^NFS`)
var nameRegexp = regexp.MustCompile(`(\d+)$`)

// VG
var viosverRegexp = regexp.MustCompile(`^AAA.VIOS.(\S+)`)
var lparnumbernameRegexp = regexp.MustCompile(`^AAA.LPARNumberName.(\d+).(\S+)`)
var aixverRegexp = regexp.MustCompile(`^AAA.AIX.(\S+)`)
var aixtlRegexp = regexp.MustCompile(`^AAA.TL.(\d+)`)
var aixmtRegexp = regexp.MustCompile(`^AAA.MachineType.IBM.(\S+)`)
//var aixcpusRegexp = regexp.MustCompile(`^BBB.*CPU in sys.(\d+)`)
var aixcpusRegexp = regexp.MustCompile(`^BBB.*lparstat.*Active.Physical.CPUs.in.system\s*:\s*(\d+)`)
var aixsmtRegexp = regexp.MustCompile(`^BBB.*smt threads.(\d+)`)
var aixcputypeRegexp = regexp.MustCompile(`^BBB.*lsconf.*Processor.Type:\s*(\w+)`)
//var aixcputypeRegexp = regexp.MustCompile(`^BBB.*lsconf.*Processor.Type:\s*([^\"]+)`)
var aixcpumodeRegexp = regexp.MustCompile(`^BBB.*lsconf.*Processor.Implementation.Mode:\s*([^\"]+)`)
//var aixfirmwareRegexp = regexp.MustCompile(`^BBB.*lsconf.*Firmware.Version:\s*([^\"]+)`)
var aixfirmwareRegexp = regexp.MustCompile(`^BBB.*lsconf.*Firmware.Version:\s*(IBM,)*([\w\.\d]+)`)

//var linuxserialRegexp = regexp.MustCompile(`^BBB.*ppc64_utils.*lscfg.*01:01\s+\w+.(\w+)`)
var linuxserialRegexp = regexp.MustCompile(`^BBB.*ppc64_utils.*lscfg.*\shost0.*(\w{7})`)
var linuxverRegexp = regexp.MustCompile(`^BBB.*\/etc\/\S*PRETTY_NAME=Q*([^\"Q*]+)`)
var linuxkernelRegexp = regexp.MustCompile(`^AAA.*Linux,(\S+),`)
var linuxmtRegexp = regexp.MustCompile(`^BBB.*ppc64_utils.*lscfg.*Model Name: (\S{8})`)
//var linuxcpsRegexp = regexp.MustCompile(`^BBB.*lscpu.*per.socket:\s+(\w+)`)
//var linuxsocketsRegexp = regexp.MustCompile(`^BBB.*lscpu.*Socket\(s\).*(\d+)`)
var linuxcpusRegexp = regexp.MustCompile(`^BBB.*ppc64_cpu.*cores.*Number.of.cores.present\W+(\d+)`)
//var linuxsmtRegexp = regexp.MustCompile(`^BBB.*ppc64_cpu.*smt.*SMT=(\w+)`)
var linuxsmtRegexp = regexp.MustCompile(`^BBB.*lscpu.*Thread.*per.core.*(\d+)`)
var linuxcputypeRegexp = regexp.MustCompile(`^BBB.*lscpu.*Model.name:\s*(\w+)`)
//var linuxfirmwareRegexp = regexp.MustCompile(`^BBB.*ppc64_utils.*lsmcode.*bmc-firmware-version-([^\"]+)`)
var linuxfirmwareRegexp = regexp.MustCompile(`^BBB.*ppc64_utils.*lsmcode.*Firmware\sis\s*([\w\.\d]+)`)
var linuxbmcfirmwareRegexp = regexp.MustCompile(`^BBB.*ppc64_utils.*lsmcode.*bmc-firmware-version-\s*([\w\.\d]+)`)

var x86cpusRegexp = regexp.MustCompile(`^AAA.x86.Cores.(\d+)`)
var x86cpumodeRegexp = regexp.MustCompile(`^AAA.x86.ModelName.*(\w{2}-\d{4})`)

var uptimeRegexp = regexp.MustCompile(`^BBB.*uptime.*up\s+([\w\s:]+)`)

var aixfcwwpnRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).World Wide Port Name\S*\s*0x(\w{16})`)
var aixfcportspeedRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Port Speed \(running\)\S*\s*(\d+)`)
var aixfcattentionRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Attention Type\S*\s*(.*)`)
var aixfclipcountRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).LIP Count\S*\s*(\d+)`)
var aixfcnoscountRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).NOS Count\S*\s*(\d+)`)
var aixfcerrframesRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Error Frames\S*\s*(\d+)`)
var aixfcdumpedframesRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Dumped Frames\S*\s*(\d+)`)
var aixfclinkfailureRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Link Failure Count\S*\s*(\d+)`)
var aixfclossofsyncRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Loss of Sync Count\S*\s*(\d+)`)
var aixfclossofsignalRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Loss of Signal\S*\s*(\d+)`)
var aixfcinvalidtxRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Invalid Tx Word Count\S*\s*(\d+)`)
var aixfcinvalidcrcRegexp = regexp.MustCompile(`^BBB.*FC\d+.(fcs\d+).Invalid CRC Count\S*\s*(\d+)`)

//Import is the entry point for subcommand nmon import
func Import(c *cli.Context) error {

	if c.Args().Len() < 1 {
		fmt.Printf("file name or directory needs to be provided\n")
		os.Exit(1)
	}

	// parsing parameters
	config := nmon2influxdblib.ParseParameters(c)

	//getting databases connections
	influxdb := config.GetDB("nmon")
	influxdbLog := config.GetLogDB()

	nmonFiles := new(nmon2influxdblib.Files)
	nmonFiles.Parse(c.Args().Slice(), config.ImportSSHUser, config.ImportSSHKey)

	tagParsers := nmon2influxdblib.ParseInputs(config.Inputs)

	var userSkipRegexp *regexp.Regexp
	if len(config.ImportSkipMetrics) > 0 {
		skipped := strings.Replace(config.ImportSkipMetrics, ",", "|", -1)
		userSkipRegexp = regexp.MustCompile(skipped)
	}

	for _, nmonFile := range nmonFiles.Valid() {

		// store the list of metrics which was logged as skipped
		LoggedSkippedMetrics := make(map[string]bool)
		var count int64
		count = 0
		nmon := InitNmon(config, nmonFile)

		if len(config.Inputs) > 0 {
			//Build tag parsing
			nmon.TagParsers = tagParsers
		}

		if nmon.Debug {
			log.Printf("Import file: %s", nmonFile.Name)
		}

		lines := nmonFile.Content()
		log.Printf("NMON file separator: %s\n", nmonFile.Delimiter)
		var last string
		filters := new(influxdbclient.Filters)
		filters.Add("file", path.Base(nmonFile.Name), "text")

		timeStamp, err := influxdbLog.ReadLastPoint("value", filters, "timestamp")
		nmon2influxdblib.CheckError(err)

		if nmon.Debug {
			log.Printf("influxdb stored timestamp: %v\n", timeStamp)
		}

		var lastTime time.Time
		if !nmon.Config.ImportForce && len(timeStamp) > 0 {
			lastTime, err = nmon.ConvertTimeStamp(timeStamp)
		} else {
			lastTime, err = nmon.ConvertTimeStamp("00:00:00,01-JAN-1900")
		}
		nmon2influxdblib.CheckError(err)

		origChecksum, err := influxdbLog.ReadLastPoint("value", filters, "checksum")
		nmon2influxdblib.CheckError(err)

		if nmon.Debug {
			log.Printf("influxdb stored checksum: %v\n", origChecksum)
		}

		ckfield := map[string]interface{}{"value": nmonFile.Checksum()}
		if !nmon.Config.ImportForce && len(origChecksum) > 0 {

			if origChecksum == nmonFile.Checksum() {
				fmt.Printf("file not changed since last import: %s\n", nmonFile.Name)
				continue
			}
		}

		//VG++
		systags := map[string]string{	"host": nmon.Hostname,
						"name": "smt",
						"mtype": nmon.MT,
						"serial": nmon.Serial,
						"SysCPU": nmon.CPUs,
						"CPUtype": nmon.CPUtype,
						"CPUmode": nmon.CPUmode,
						"FWlevel": nmon.FW,
						"os": nmon.OS,
						"osver": nmon.OSver,
						"osrelease": nmon.OStl,
						"uptime": nmon.uptime,
						"lparnr": nmon.LPARnr,
						"lparname": nmon.LPARname}

		// try to convert smt string to integer
		smtfloat := 1.0
		converted, parseErr := strconv.ParseFloat(nmon.SMT, 64)
                if parseErr != nil || math.IsNaN(converted) {
                        //if not working, skip to next value. We don't want text values in InfluxDB.
			smtfloat = 1.0
		} else {
			smtfloat = converted
		}

		//type Threads map[string]Thread
		type Thread struct {
			MaxThreads float64
			Cmd string
		}
		threads := make(map[string]Thread)
		const TopThreads = 5
		var currTimestamp = ""
		//var maxThreads = 0.0
                //var thPid = ""
                //var thCmd = ""
		//VG--
		for _, line := range lines {

			if cpuallRegexp.MatchString(line) && !config.ImportAllCpus {
				continue
			}

			if diskallRegexp.MatchString(line) && config.ImportSkipDisks {
				continue
			}

			if skipRegexp.MatchString(line) {
				continue
			}

			if statsRegexp.MatchString(line) {
				matched := statsRegexp.FindStringSubmatch(line)
				elems := strings.Split(line, nmonFile.Delimiter)
				name := elems[0]
				//VG++
				measurement := ""
                                if nfsRegexp.MatchString(name) || cpuallRegexp.MatchString(name) {
					measurement = name
				} else {
					measurement = nameRegexp.ReplaceAllString(name, "")
				}
				//VG ---

				//VG++  maxthreads
                                if topRegexp.MatchString(line) {
                                      matched := topRegexp.FindStringSubmatch(line)
				      if currTimestamp != matched[1] {
					      if len(currTimestamp) > 0 {

						      timeStr, getErr := nmon.GetTimeStamp(currTimestamp)
						      if getErr != nil {
                                                              continue
                                                      }
                                                      timestamp, convErr := nmon.ConvertTimeStamp(timeStr)
                                                      nmon2influxdblib.CheckError(convErr)

						      //tags := map[string]string{"host": nmon.Hostname, "name": "maxThreads", "pid": thPid, "command": thCmd}
                                                      //if len(nmon.Serial) > 0 {
						      //      tags["serial"] = nmon.Serial
                                                      //}
                                                      //send integer if it worked
                                                      //field := map[string]interface{}{"value": maxThreads}
                                                      //influxdb.AddPoint("THREAD", timestamp, field, tags)

						      for key, val := range threads {
							      tags := map[string]string{"host": nmon.Hostname, "name": "maxThreads", "pid": key, "command": val.Cmd}
							      if len(nmon.Serial) > 0 {
                                                                  tags["serial"] = nmon.Serial
                                                              }
                                                              //send integer if it worked
                                                              field := map[string]interface{}{"value": val.MaxThreads}
                                                              influxdb.AddPoint("THREADS", timestamp, field, tags)
						      }
					      }
					      //maxThreads = 0.0
					      currTimestamp = matched[1]
					      //thPid = ""
					      //thCmd = ""
					      threads = make(map[string]Thread)
			              }


                                      // elems := strings.Split(line, nmonFile.Delimiter)
                                      // name := elems[0]

                                      if len(elems) < 14 {
                                            log.Println(elems)
                                            continue
                                      }
				      // try to convert string to integer
                                      converted, parseErr := strconv.ParseFloat(elems[6], 64)
                                      if parseErr != nil {
                                            //if not working, skip to next value. We don't want text values in InfluxDB.
                                            continue
                                      }
				      //if converted > maxThreads {
				      //      maxThreads = converted
				      //      thPid = elems[1]
				      //      thCmd = elems[13]
				      //}

				      if len(threads) < TopThreads {
					      t1 := Thread{converted, elems[13]}
					      threads[elems[1]] = t1
				      } else {
					      // p, t := getMinTh(threads)
					      minPid := "1"
					      minThr := 0.0
					      for k, v := range threads {
						      if (minThr <= 0.0) || (v.MaxThreads < minThr) {
							      minPid = k
							      minThr = v.MaxThreads
						      }
					      }
					      if converted > minThr {
					            delete (threads, minPid)
						    t1 := Thread{converted, elems[13]}
						    threads[elems[1]] = t1
					      }
				      }

				      //continue
                                }  // if topRegexp.MatchString(line)

				//VG-- maxthreads

				if len(config.ImportSkipMetrics) > 0 {
					if userSkipRegexp.MatchString(name) {
						if nmon.Debug {
							if !LoggedSkippedMetrics[name] {
								log.Printf("metric skipped : %s\n", name)
								LoggedSkippedMetrics[name] = true
							}
						}
						continue
					}
				}

				timeStr, getErr := nmon.GetTimeStamp(matched[1])
				if getErr != nil {
					continue
				}
				last = timeStr
				timestamp, convErr := nmon.ConvertTimeStamp(timeStr)
				nmon2influxdblib.CheckError(convErr)
				if timestamp.Before(lastTime) && !nmon.Config.ImportForce {
					continue
				}

				for i, value := range elems[2:] {
					if len(nmon.DataSeries[name].Columns) < i+1 {
						if nmon.Debug {
							log.Printf(line)
							log.Printf("Entry added position %d in serie %s since nmon start: skipped\n", i+1, name)
						}
						continue
					}
					column := nmon.DataSeries[name].Columns[i]
					tags := map[string]string{"host": nmon.Hostname, "name": column}
					// try to convert string to integer
					converted, parseErr := strconv.ParseFloat(value, 64)
					if parseErr != nil || math.IsNaN(converted) {
						//if not working, skip to next value. We don't want text values in InfluxDB.
						continue
					}

					//send integer if it worked
					field := map[string]interface{}{"value": converted}

					//VG++
					//measurement := ""
					//if nfsRegexp.MatchString(name) || cpuallRegexp.MatchString(name) {
					//	measurement = name
					//} else {
					//	measurement = nameRegexp.ReplaceAllString(name, "")
					//}
					//VG + 
					if measurement == "CPU_ALL" {
						if len(nmon.MT) > 0 {
							tags["mtype"] = nmon.MT
						}
						if len(nmon.Serial) > 0 {
							tags["serial"] = nmon.Serial
						}
						if len(nmon.SMT) > 0 {
							tags["smt"] = nmon.SMT
						}
						if len(nmon.CPUs) > 0 {
							tags["cpus_in_sys"] = nmon.CPUs
						}
					}
					if measurement == "MEM" {
						if len(nmon.MT) > 0 {
                                                        tags["mtype"] = nmon.MT
                                                }
                                                if len(nmon.Serial) > 0 {
                                                        tags["serial"] = nmon.Serial
                                                }
					}

					// Checking additional tagging
					for key, value := range tags {
						if _, ok := nmon.TagParsers[measurement][key]; ok {
							for _, tagParser := range nmon.TagParsers[measurement][key] {
								if tagParser.Regexp.MatchString(value) {
									tags[tagParser.Name] = tagParser.Value
								}
							}
						}

						if _, ok := nmon.TagParsers["_ALL"][key]; ok {
							for _, tagParser := range nmon.TagParsers["_ALL"][key] {
								if tagParser.Regexp.MatchString(value) {
									tags[tagParser.Name] = tagParser.Value
								}
							}
						}

					}
					influxdb.AddPoint(measurement, timestamp, field, tags)

					if influxdb.PointsCount() >= 5000 {
						err = influxdb.WritePoints()
						nmon2influxdblib.CheckError(err)
						count += influxdb.PointsCount()
						influxdb.ClearPoints()
						fmt.Printf("#")
					}
				}  // for i, value := range elems[2:]
				if measurement == "CPU_ALL" {

					// write SYSINFO measurement
					sysfield := map[string]interface{}{"value": smtfloat}

					// Checking additional tagging
                                        for key, value := range systags {
                                                if _, ok := nmon.TagParsers["SYSINFO"][key]; ok {
                                                        for _, tagParser := range nmon.TagParsers["SYSINFO"][key] {
                                                                if tagParser.Regexp.MatchString(value) {
                                                                        systags[tagParser.Name] = tagParser.Value
                                                                }
                                                        }
                                                }

                                                if _, ok := nmon.TagParsers["_ALL"][key]; ok {
                                                        for _, tagParser := range nmon.TagParsers["_ALL"][key] {
                                                                if tagParser.Regexp.MatchString(value) {
                                                                        systags[tagParser.Name] = tagParser.Value
                                                                }
                                                        }
                                                }

                                        }

					influxdb.AddPoint("SYSINFO", timestamp, sysfield, systags)
				}  // if measurement == "CPU_ALL"
				if measurement == "FCREAD" {
					// write FC adapter statistics report
					for dev, devval := range nmon.FCs {
						FCfields := map[string]string{
							"speed": devval.speed,
							"lipcnt": devval.lipcnt,
							"noscnt": devval.noscnt,
							"errframe": devval.errframe,
							"dumpframe": devval.dumpframe,
							"linkfail": devval.linkfail,
							"losssync": devval.losssync,
							"losssig": devval.losssig,
							"invtx": devval.invtx,
							"invcrc": devval.invcrc}

						FCtags := map[string]string{
							"name": "",
							"dev": dev,
							"wwpn": devval.wwpn,
							"atttype": devval.att,
							"host": nmon.Hostname,
							"mtype": nmon.MT,
							"serial": nmon.Serial,
							"lparnr": nmon.LPARnr,
                                                        "lparname": nmon.LPARname}

						for fname, fval := range FCfields {
							// 
                                                        // try to convert smt string to integer
                                                        fieldfloat := 0.0
                                                        converted, parseErr := strconv.ParseFloat(fval, 64)
                                                        if parseErr != nil || math.IsNaN(converted) {
                                                                fieldfloat = 1.0
                                                        } else {
                                                                fieldfloat = converted
							}
							FCfield := map[string]interface{}{"value": fieldfloat}
							FCtags["name"] = fname

                                                        // Checking additional tagging
                                                        for key, value := range systags {
                                                                if _, ok := nmon.TagParsers["FCSTAT"][key]; ok {
                                                                        for _, tagParser := range nmon.TagParsers["FCSTAT"][key] {
                                                                                if tagParser.Regexp.MatchString(value) {
                                                                                        FCtags[tagParser.Name] = tagParser.Value
                                                                                }
                                                                        }
                                                                }

                                                                if _, ok := nmon.TagParsers["_ALL"][key]; ok {
                                                                        for _, tagParser := range nmon.TagParsers["_ALL"][key] {
                                                                                if tagParser.Regexp.MatchString(value) {
                                                                                        FCtags[tagParser.Name] = tagParser.Value
                                                                                }
                                                                        }
                                                                }
                                                        }

							influxdb.AddPoint("FCSTAT", timestamp, FCfield, FCtags)
						} // for field := range FCfields
					} // for k := range nmon.FCs
				} // if measurement == "FCREAD"
			}  // if statsRegexp.MatchString(line)

			if topRegexp.MatchString(line) {
				matched := topRegexp.FindStringSubmatch(line)

				elems := strings.Split(line, nmonFile.Delimiter)
				name := elems[0]
				if len(config.ImportSkipMetrics) > 0 {
					if userSkipRegexp.MatchString(name) {
						if nmon.Debug {
							if !LoggedSkippedMetrics[name] {
								log.Printf("metric skipped : %s\n", name)
								LoggedSkippedMetrics[name] = true
							}
						}
						continue
					}
				}

				timeStr, getErr := nmon.GetTimeStamp(matched[1])
				if getErr != nil {
					continue
				}

				timestamp, convErr := nmon.ConvertTimeStamp(timeStr)
				nmon2influxdblib.CheckError(convErr)

				if len(elems) < 14 {
					log.Printf("error TOP import:")
					log.Println(elems)
					continue
				}

				for i, value := range elems[3:12] {
					column := nmon.DataSeries["TOP"].Columns[i]

					var wlmclass string
					if len(elems) < 15 {
						wlmclass = "none"
					} else {
						wlmclass = elems[14]
					}

					tags := map[string]string{"host": nmon.Hostname, "name": column, "pid": elems[1], "command": elems[13], "wlm": wlmclass}

					if len(nmon.Serial) > 0 {
						tags["serial"] = nmon.Serial
					}

					// try to convert string to integer
					converted, parseErr := strconv.ParseFloat(value, 64)
					if parseErr != nil {
						//if not working, skip to next value. We don't want text values in InfluxDB.
						continue
					}

					//send integer if it worked
					field := map[string]interface{}{"value": converted}

					influxdb.AddPoint("TOP", timestamp, field, tags)

					if influxdb.PointsCount() == 10000 {
						err = influxdb.WritePoints()
						nmon2influxdblib.CheckError(err)
						count += influxdb.PointsCount()
						influxdb.ClearPoints()
						fmt.Printf("#")
					}
				}
			}  // if topRegexp.MatchString(line)
		}  // for _, line := range lines
		// flushing remaining data
		influxdb.WritePoints()
		count += influxdb.PointsCount()
		fmt.Printf("\nFile %s imported : %d points !\n", nmonFile.Name, count)
		if config.ImportBuildDashboard {
			DashboardFile(config, nmonFile.Name)
		}

		if len(last) > 0 {
			field := map[string]interface{}{"value": last}
			tag := map[string]string{"file": path.Base(nmonFile.Name)}
			lasttime, _ := nmon.ConvertTimeStamp("now")
			influxdbLog.AddPoint("timestamp", lasttime, field, tag)
			influxdbLog.AddPoint("checksum", lasttime, ckfield, tag)
			err = influxdbLog.WritePoints()
			nmon2influxdblib.CheckError(err)
		}
	}

	return nil
}
