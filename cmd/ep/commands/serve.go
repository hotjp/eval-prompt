package commands

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动本地 HTTP 服务（包含 Web UI）",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		host, _ := cmd.Flags().GetString("host")
		noBrowser, _ := cmd.Flags().GetBool("no-browser")

		addr := fmt.Sprintf("%s:%d", host, port)
		fmt.Printf("启动服务: http://%s\n", addr)

		if !noBrowser {
			fmt.Println("打开浏览器...")
		}

		return http.ListenAndServe(addr, nil)
	},
}

func init() {
	serveCmd.Flags().Int("port", 8080, "服务端口")
	serveCmd.Flags().String("host", "127.0.0.1", "监听地址")
	serveCmd.Flags().Bool("no-browser", false, "不自动打开浏览器")
}
