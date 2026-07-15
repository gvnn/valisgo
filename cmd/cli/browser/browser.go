package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Open(url string) {
	var err error

	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/c", "start", "", url).Start()

	case "darwin":
		err = exec.Command("open", url).Start()

	case "android":
		err = exec.Command("am", "start", "-a", "android.intent.action.VIEW", "-d", url).Start()

	default:
		browsers := []string{
			"wslview",
			"xdg-open",
			"sensible-browser",
			"firefox",
			"google-chrome",
			"chromium",
			"chromium-browser",
			"safari",
			"opera",
			"epiphany",
		}

		success := false
		for _, browser := range browsers {
			if _, lookErr := exec.LookPath(browser); lookErr == nil {
				if runErr := exec.Command(browser, url).Start(); runErr == nil {
					success = true
					break
				}
			}
		}

		if !success {
			err = fmt.Errorf("no suitable browser found")
		}
	}

	if err != nil {
		fmt.Printf("Could not open browser automatically. Please open this URL manually:\n%s\n", url)
	}
}
