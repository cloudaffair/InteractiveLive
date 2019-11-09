package main

import(
	"github.com/gin-gonic/gin"
	"common"
	"time"
	"os"
	"ila"
	"fmt"
)

func main() {
	config, err := common.GetConfig()
	if err != nil {
		fmt.Errorf("Configuration fetch failed %s", err)
		time.Sleep(time.Second * 5) // allow time for logger threads to write errors
		os.Exit(1)
	}

	fmt.Println("Interactive Live service Startup")

	ilaApp, err := ila.NewIla(config)
	if err != nil {
		fmt.Errorf("App init failed %s", err)

		time.Sleep(time.Second * 2) // allow time for logger threads to write errors
		os.Exit(1)
	}
	//turn off gin messages
	gin.SetMode(gin.ReleaseMode)
	ilaApp.Run(gin.New())
}
