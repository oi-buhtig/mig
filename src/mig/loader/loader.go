// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

// The MIG loader is a simple bootstrapping tool for MIG. It can be scheduled
// to run on a host system and download the newest available version of the
// agent. If the loader identifies a newer version of the agent available, it
// will download the required files from the API, replace the existing files,
// and notify any existing agent it should terminate.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/jvehent/cljs"
	"io/ioutil"
	"mig"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
)

var apiManifest *mig.ManifestResponse

func initializeHaveBundle() ([]mig.BundleDictionaryEntry, error) {
	ret, err := mig.GetHostBundle()
	if err != nil {
		return nil, err
	}
	ret, err = mig.HashBundle(ret)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "initializeHaveBundle() -> Initialized\n")
	for _, x := range ret {
		fmt.Fprintf(os.Stderr, "%v %v -> %v\n", x.Name, x.Path, x.SHA256)
	}
	return ret, nil
}

func requestManifest() error {
	murl := APIURL + "manifest"
	fmt.Fprintf(os.Stderr, "requestManifest() -> requesting manifest from %v\n", murl)

	mparam := mig.ManifestParameters{}
	mparam.OS = runtime.GOOS
	mparam.Arch = runtime.GOARCH
	mparam.Operator = TAGS.Operator
	buf, err := json.Marshal(mparam)
	if err != nil {
		return err
	}
	mstring := string(buf)
	data := url.Values{"parameters": {mstring}}
	r, err := http.NewRequest("POST", murl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		return err
	}

	// Extract our manifest from the response.
	manifest, err := valueToManifest(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		return err
	}
	apiManifest = &manifest

	return nil
}

func valueToManifest(v interface{}) (m mig.ManifestResponse, err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &m)
	return
}

func main() {
	runtime.GOMAXPROCS(1)

	// Get our current status from the file system.
	_, err := initializeHaveBundle()
	if err != nil {
		fmt.Fprintf(os.Stderr, "main() -> %v\n", err)
		os.Exit(1)
	}

	// Retrieve our manifest from the API.
	err = requestManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "main() -> %v\n", err)
		os.Exit(1)
	}
}