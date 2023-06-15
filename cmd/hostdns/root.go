package hostdns

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	dnsserver "github.com/ahanwhite/hostsdns/pkg/server"
)

type runFlags struct {
	hostsfile string
	port      int
}

var runFlag runFlags

var longRootCmdDescription = `hostsdns is a dns server that implements local resolution functions similar to /etc/hosts. 
Using hostsdns can support the local resolution function and avoid the resolution failure caused by tampering of /etc/hosts`

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hostsdns",
	Short: "A dns server that implements local resolution functions similar to /etc/hosts.",
	Long:  longRootCmdDescription,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err      error
			hostfile = runFlag.hostsfile
			port     = runFlag.port
		)

		// 检测文件是否存在
		if _, err := os.Stat(hostfile); os.IsNotExist(err) {
			// 文件不存在，创建文件
			file, err := os.Create(hostfile)
			if err != nil {
				log.Printf("Unable to create the file %s: %v\n", hostfile, err)
				return err
			}
			defer file.Close()

			log.Printf("File %s created successfully\n", hostfile)
		} else if err != nil {
			// 发生其他错误
			fmt.Printf("无法访问文件 %s: %v\n", hostfile, err)
			return err
		}

		// 初始化 DNS 服务器
		server := dnsserver.NewDnsServer(hostfile, port)

		if err = server.InitServer(); err != nil {
			log.Fatal("Failed to initialize DNS server:", err)
			return err
		}

		if err = startServer(server); err != nil {
			log.Fatal("Failed to start DNS server:", err)
			return err
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println("hostdns err:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&runFlag.hostsfile, "hostfile", "/etc/hostdns", "hostfile for dnsserver (default is /etc/hostdns)")
	rootCmd.PersistentFlags().IntVarP(&runFlag.port, "port", "p", 53, "port for dns server using udp protocol (default is 53)")
	rootCmd.DisableAutoGenTag = true
}

func startServer(ds dnsserver.Interface) error {
	// 监听 Hosts 文件变动并重新加载
	go func() {
		for range time.Tick(time.Second) {
			// 检查 Hosts 文件是否发生变动
			modified, err := ds.IsHostsFileModified()
			if err != nil {
				log.Println("Failed to check Hosts file modification:", err)
				continue
			}

			if modified {
				// Hosts 文件发生变动，重新加载
				err := ds.LoadHostsFile()
				if err != nil {
					log.Println("Failed to reload Hosts file:", err)
					continue
				}

				log.Println("Hosts file reloaded")
			}
		}
	}()

	// 启动 DNS 服务器
	err := ds.Start()
	if err != nil {
		log.Fatal("Failed to start DNS server:", err)
		return err
	}
	return nil
}
