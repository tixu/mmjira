package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/pkg/profile"
)

func dumpRequest(r *http.Request) (err error) {

	dump, err := httputil.DumpRequest(r, true)

	if err != nil {
		return err
	}
	if Config.Debug {

		tmpfile, err := ioutil.TempFile(Config.DumpDir, "example")
		if err != nil {
			return err
		}
		tmpfile.Write(dump)
		tmpfile.Close()
	}
	return nil
}

func activateProfiline(c InstanceConfig) {
	// activating
	switch c.Profile {
	case "cpu":
		defer profile.Start(profile.ProfilePath(c.ProfileDir), profile.CPUProfile).Stop()
	case "mem":
		defer profile.Start(profile.ProfilePath(c.ProfileDir), profile.MemProfile).Stop()
	case "block":
		defer profile.Start(profile.ProfilePath(c.ProfileDir), profile.BlockProfile).Stop()
	default:
		// do nothing
	}

}
