package utils

import (
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/pkg/profile"
)

func DumpRequest(r *http.Request, dir string) (err error) {

	dump, err := httputil.DumpRequest(r, true)

	if err != nil {
		return err
	}

	tmpfile, err := ioutil.TempFile(dir, "example")
	if err != nil {
		return err
	}
	tmpfile.Write(dump)
	tmpfile.Close()

	return nil

}

func activateProfiline(profileType string, dir string) {
	// activating
	switch profileType {
	case "cpu":
		defer profile.Start(profile.ProfilePath(dir), profile.CPUProfile).Stop()
	case "mem":
		defer profile.Start(profile.ProfilePath(dir), profile.MemProfile).Stop()
	case "block":
		defer profile.Start(profile.ProfilePath(dir), profile.BlockProfile).Stop()
	default:
		// do nothing
	}

}
