package swagger2

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grokify/gotilla/io/ioutilmore"
	"github.com/pkg/errors"
)

var jsonFileRx = regexp.MustCompile(`(?i)\.json\s*$`)

func MergeDirectory(dir string) (Specification, error) {
	fileInfos, err := ioutilmore.DirEntriesRxSizeGt0(dir, ioutilmore.File, jsonFileRx)
	if err != nil {
		return Specification{}, err
	}
	if len(fileInfos) == 0 {
		return Specification{}, fmt.Errorf("No JSON files found in directory [%s]", dir)
	}
	var specMaster Specification
	for i, fi := range fileInfos {
		thisSpecFilepath := filepath.Join(dir, fi.Name())
		thisSpec, err := ReadSwagger2Spec(thisSpecFilepath)
		if err != nil {
			return specMaster, err
		}
		if i == 0 {
			specMaster = thisSpec
		} else {
			specMaster = Merge(specMaster, thisSpec)
		}
	}
	return specMaster, nil
}

func Merge(specMaster, specExtra Specification) Specification {
	specMaster = MergeTags(specMaster, specExtra)
	specMaster = MergePaths(specMaster, specExtra)
	return MergeDefinitions(specMaster, specExtra)
}

func MergeTags(specMaster, specExtra Specification) Specification {
	tagsMap := map[string]int{}
	for _, tag := range specMaster.Tags {
		tagsMap[tag.Name] = 1
	}
	for _, tag := range specExtra.Tags {
		tag.Name = strings.TrimSpace(tag.Name)
		if _, ok := tagsMap[tag.Name]; !ok {
			specMaster.Tags = append(specMaster.Tags, tag)
		}
	}
	return specMaster
}

func MergePaths(specMaster, specExtra Specification) Specification {
	for url, path := range specExtra.Paths {
		specMaster.Paths[url] = path
	}
	return specMaster
}

func MergeDefinitions(specMaster, specExtra Specification) Specification {
	for definitionName, def := range specExtra.Definitions {
		specMaster.Definitions[definitionName] = def
	}
	return specMaster
}

func WriteFileDirMerge(outfile, inputDir string, perm os.FileMode) error {
	spec, err := MergeDirectory(inputDir)
	if err != nil {
		return errors.Wrap(err, "E_MERGE_DIRECTORY_FAILED")
	}

	err = ioutilmore.WriteFileJSON(outfile, spec, perm, "", "  ")
	if err != nil {
		return errors.Wrap(err, "E_WRITE_FAILED")
	}
	return nil
}
