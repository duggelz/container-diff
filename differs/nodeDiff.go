package differs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/container-diff/utils"
	"github.com/golang/glog"
)

type NodeAnalyzer struct {
}

// NodeDiff compares the packages installed by apt-get.
func (a NodeAnalyzer) Diff(image1, image2 utils.Image) (utils.Result, error) {
	diff, err := multiVersionDiff(image1, image2, a)
	return diff, err
}

func (a NodeAnalyzer) Analyze(image utils.Image) (utils.Result, error) {
	analysis, err := multiVersionAnalysis(image, a)
	return analysis, err
}

func (a NodeAnalyzer) getPackages(image utils.Image) (map[string]map[string]utils.PackageInfo, error) {
	path := image.FSPath
	packages := make(map[string]map[string]utils.PackageInfo)
	if _, err := os.Stat(path); err != nil {
		// path provided invalid
		return packages, err
	}
	layerStems, err := buildNodePaths(path)
	if err != nil {
		glog.Warningf("Error building JSON paths at %s: %s\n", path, err)
		return packages, err
	}

	for _, modulesDir := range layerStems {
		packageJSONs, _ := utils.BuildLayerTargets(modulesDir, "package.json")
		for _, currPackage := range packageJSONs {
			if _, err := os.Stat(currPackage); err != nil {
				// package.json file does not exist at this target path
				continue
			}
			packageJSON, err := readPackageJSON(currPackage)
			if err != nil {
				glog.Warningf("Error reading package JSON at %s: %s\n", currPackage, err)
				return packages, err
			}
			// Build PackageInfo for this package occurence
			var currInfo utils.PackageInfo
			currInfo.Version = packageJSON.Version
			packagePath := strings.TrimSuffix(currPackage, "package.json")
			currInfo.Size = utils.GetSize(packagePath)
			mapPath := strings.Replace(packagePath, path, "", 1)
			// Check if other package version already recorded
			if _, ok := packages[packageJSON.Name]; !ok {
				// package not yet seen
				infoMap := make(map[string]utils.PackageInfo)
				infoMap[mapPath] = currInfo
				packages[packageJSON.Name] = infoMap
				continue
			}
			packages[packageJSON.Name][mapPath] = currInfo

		}
	}
	return packages, nil
}

type nodePackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func buildNodePaths(path string) ([]string, error) {
	globalPaths := filepath.Join(path, "node_modules")
	localPath := filepath.Join(path, "usr/local/lib/node_modules")
	return []string{globalPaths, localPath}, nil
}

func readPackageJSON(path string) (nodePackage, error) {
	var currPackage nodePackage
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return currPackage, err
	}
	err = json.Unmarshal(jsonBytes, &currPackage)
	if err != nil {
		return currPackage, err
	}
	return currPackage, err
}
