// nmon2influxdb
// import nmon data in InfluxDB

// author: adejoux@djouxtech.net

package main

import (
	"log"
	"os"
	//"fmt"

	"github.com/adejoux/nmon2influxdb/hmc"
	"github.com/adejoux/nmon2influxdb/nmon"
	"github.com/adejoux/nmon2influxdb/nmon2influxdblib"
	"github.com/urfave/cli/v2"
)

func main() {
	config := nmon2influxdblib.InitConfig()

	// VG: if config file provided by the CLI parameters
	configFile  := ""
    configIndex := -1

    myArgs := os.Args
    for i, arg := range myArgs {
        if configIndex >= 0 {
            configFile = arg
            break
        } else {
            if arg == "--configfile" {
                configIndex = i
            }
        }
    }
    if configIndex >= 0 {
        if configFile =="" {
            log.Fatalf("Error, Use: --configfile <config file>\n")
            os.Exit(1)
        }
        //myArgs = config.removeArgs(myArgs, configIndex)
    }
	cfgfile := config.LoadCfgFile(configFile)
	// VG
	// cfgfile := config.LoadCfgFile()
	if len(config.DebugFile) > 0 {
		debugFile, err := os.OpenFile(config.DebugFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer debugFile.Close()
		log.SetOutput(debugFile)

	}

	// moved to the end of file log.Printf("Using configuration file %s\n", cfgfile)

	// cannot set values directly for boolean flags
	if config.DashboardWriteFile {
		os.Setenv("NMON2INFLUXDB_DASHBOARD_TO_FILE", "true")
	}

	if config.ImportSkipDisks {
		os.Setenv("NMON2INFLUXDB_SKIP_DISKS", "true")
	}

	if config.ImportAllCpus {
		os.Setenv("NMON2INFLUXDB_ADD_ALL_CPUS", "true")
	}

	if config.ImportBuildDashboard {
		os.Setenv("NMON2INFLUXDB_BUILD_DASHBOARD", "true")
	}

	if config.ImportForce {
		os.Setenv("NMON2INFLUXDB_FORCE", "true")
	}

	if config.InfluxdbSecure {
		os.Setenv("NMON2INFLUXDB_SECURE", "true")
	}
	if config.InfluxdbSkipCertCheck {
		os.Setenv("NMON2INFLUXDB_SKIP_CERT_CHECK", "true")
	}
	if len(config.ImportSkipMetrics) > 0 {
		os.Setenv("NMON2INFLUXDB_SKIP_METRICS", config.ImportSkipMetrics)
	}

	if len(config.HMCServer) > 0 {
		os.Setenv("NMON2INFLUXDB_HMC_SERVER", config.HMCServer)
	}

	if len(config.ImportSkipMetrics) > 0 {
		os.Setenv("NMON2INFLUXDB_HMC_USER", config.HMCServer)
	}

	app := cli.NewApp()
	app.Name = "nmon2influxdb"
	app.Usage = "upload NMON stats to InfluxDB database"
	app.Version = "2.1.7_vg6.4"
	app.Commands = []*cli.Command{
		{
			Name:  "import",
			Usage: "import nmon files",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "skip_metrics",
					Usage:   "skip metrics",
					EnvVars: []string{"NMON2INFLUXDB_SKIP_METRICS"},
				},
				&cli.BoolFlag{
					Name:    "nodisks",
					Aliases: []string{"nd"},
					Usage:   "skip disk metrics",
					EnvVars: []string{"NMON2INFLUXDB_SKIP_DISKS"},
				},
				&cli.BoolFlag{
					Name:    "cpus",
					Aliases: []string{"c"},
					Usage:   "add per cpu metrics",
					EnvVars: []string{"NMON2INFLUXDB_ADD_ALL_CPU"},
				},
				&cli.BoolFlag{
					Name:    "build",
					Aliases: []string{"b"},
					Usage:   "build dashboard",
					EnvVars: []string{"NMON2INFLUXDB_BUILD_DASHBOARD"},
				},
				&cli.BoolFlag{
					Name:    "force",
					Aliases: []string{"f"},
					Usage:   "force import",
					EnvVars: []string{"NMON2INFLUXDB_FORCE"},
				},
				&cli.StringFlag{
					Name:  "log_database",
					Usage: "influxdb database used to log imports",
					Value: config.ImportLogDatabase,
				},
				&cli.StringFlag{
					Name:  "log_retention",
					Usage: "import log retention",
					Value: config.ImportLogRetention,
				},
				//&cli.StringFlag{
				//	Name:  "configfile",
				//	Usage: "config file",
				//	Destination: &configFile,
				//},
			},
			Action: nmon.Import,
		},
		{
			Name:  "dashboard",
			Usage: "generate a dashboard from a nmon file or template",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "file",
					Aliases: []string{"f"},
					Usage:   "generate Grafana dashboard file",
					EnvVars: []string{"NMON2INFLUXDB_DASHBOARD_TO_FILE"},
				},
				&cli.StringFlag{
					Name:  "guser",
					Usage: "grafana user",
					Value: config.GrafanaUser,
				},
				&cli.StringFlag{
					Name:    "gpassword",
					Aliases: []string{"gpass"},
					Usage:   "grafana password",
					Value:   config.GrafanaPassword,
				},
				&cli.StringFlag{
					Name:  "gaccess",
					Usage: "grafana datasource access mode : direct or proxy",
					Value: config.GrafanaAccess,
				},
				&cli.StringFlag{
					Name:  "gurl",
					Usage: "grafana url",
					Value: config.GrafanaURL,
				},
				&cli.StringFlag{
					Name:  "datasource",
					Usage: "grafana datasource",
					Value: config.GrafanaDatasource,
				},
			},
			Action: nmon.Dashboard,
		},
		{
			Name:  "stats",
			Usage: "generate stats from a InfluxDB metric",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "metric",
					Aliases: []string{"m"},
					Usage:   "mandatory metric for stats",
				},
				&cli.StringFlag{
					Name:    "statshost",
					Aliases: []string{"s"},
					Usage:   "host metrics",
					Value:   config.StatsHost,
				},
				&cli.StringFlag{
					Name:    "from",
					Aliases: []string{"f"},
					Usage:   "from date",
					Value:   config.StatsFrom,
				},
				&cli.StringFlag{
					Name:    "to",
					Aliases: []string{"t"},
					Usage:   "to date",
					Value:   config.StatsTo,
				},
				&cli.StringFlag{
					Name:  "sort",
					Usage: "field to perform sort",
					Value: config.StatsSort,
				},
				&cli.IntFlag{
					Name:  "limit,l",
					Usage: "limit the output",
					Value: config.StatsLimit,
				},
				&cli.StringFlag{
					Name:  "filter",
					Usage: "specify a filter on fields",
					Value: config.StatsFilter,
				},
			},
			Action: nmon.Stat,
		},
		{
			Name:  "list",
			Usage: "list InfluxDB metrics or measurements",
			Subcommands: []*cli.Command{
				{
					Name:  "measurement",
					Usage: "list InfluxDB measurements",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "host",
							Usage: "only for specified host",
						},
						&cli.StringFlag{
							Name:    "filter",
							Aliases: []string{"f"},
							Usage:   "filter measurement",
						},
					},
					Action: nmon.ListMeasurement,
				},
			},
		},
		{
			Name:  "hmc",
			Usage: "load hmc data",
			Subcommands: []*cli.Command{
				{
					Name:   "import",
					Usage:  "import HMC PCM data",
					Action: hmc.Import,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:    "hmc",
							Usage:   "HMC server",
							EnvVars: []string{"NMON2INFLUXDB_HMC_SERVER"},
						},
						&cli.StringFlag{
							Name:  "hmcuser",
							Usage: "HMC user",
							Value: config.HMCUser,
						},
						&cli.StringFlag{
							Name:  "hmcpass",
							Usage: "HMC password",
							Value: config.HMCPassword,
						},
						&cli.StringFlag{
							Name:  "managed_system,m",
							Usage: "only import from this managed system",
							Value: config.HMCManagedSystem,
						},
						&cli.BoolFlag{
							Name:  "managed_system-only,sys-only",
							Usage: "skip partition metrics",
						},
						&cli.IntFlag{
							Name:  "samples",
							Usage: "import latest <value> samples",
							Value: config.HMCSamples,
						},
						&cli.IntFlag{
							Name:  "timeout",
							Usage: "HMC connection timeout",
							Value: config.HMCTimeout,
						},
					},
				},
			},
		},
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "configfile",
			Usage: "config file",
			Destination: &configFile,
		},
		&cli.StringFlag{
			Name:  "server,s",
			Usage: "InfluxDB server and port",
			Value: config.InfluxdbServer,
		},
		&cli.StringFlag{
			Name:  "port,p",
			Usage: "InfluxDB port",
			Value: config.InfluxdbPort,
		},
		&cli.BoolFlag{
			Name:    "secure",
			Usage:   "use ssl for InfluxDB",
			EnvVars: []string{"NMON2INFLUXDB_SECURE"},
		},
		&cli.BoolFlag{
			Name:    "skip_cert_check",
			Usage:   "skip cert check for ssl connzction to InfluxDB",
			EnvVars: []string{"NMON2INFLUXDB_SKIP_CERT_CHECK"},
		},
		&cli.StringFlag{
			Name:  "db,d",
			Usage: "InfluxDB database",
			Value: config.InfluxdbDatabase,
		},
		&cli.StringFlag{
			Name:  "user,u",
			Usage: "InfluxDB administrator user",
			Value: config.InfluxdbUser,
		},
		&cli.StringFlag{
			Name:  "pass",
			Usage: "InfluxDB administrator pass",
			Value: config.InfluxdbPassword,
		},
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "debug mode",
			EnvVars: []string{"NMON2INFLUXDB_DEBUG"},
		},
		&cli.StringFlag{
			Name:  "debug-file",
			Usage: "debug file",
			Value: config.DebugFile,
		},
		&cli.StringFlag{
			Name:  "tz,t",
			Usage: "timezone",
			Value: config.Timezone,
		},
	}
	app.Authors = []*cli.Author{{Name: "Alain Dejoux", Email: "adejoux@djouxtech.net"},
				    {Name: "Valery Grusdev", Email: "valery@grusdev.com"}}

	log.Printf("%s-%s\n", app.Name, app.Version)				
	log.Printf("Using configuration file %s\n", cfgfile)
	app.Run(os.Args)

}
