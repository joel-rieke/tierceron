package utils

import (
	"fmt"
	"io/ioutil"
	"os"

	"bitbucket.org/dexterchaney/whoville/vault-helper/kv"
)

//ConfigTemplates takes a file directory to read templates from and a directory to write templates to and configures the templates.
func ConfigTemplates(dir string, endDir string, modifier *kv.Modifier, dataPaths ...string) {
	//get files from directory
	templatePaths, endPaths := getDirFiles(dir, endDir)
	//configure each template in directory
	for i, templatePath := range templatePaths {
		ConfigTemplate(modifier, templatePath, endPaths[i], dataPaths...)
	}
	fmt.Println("templates configured and written to ", endDir)
	//config each template in directory
}

func getDirFiles(dir string, endDir string) ([]string, []string) {
	files, err := ioutil.ReadDir(dir)
	filePaths := []string{}
	endPaths := []string{}
	if err != nil {
		//this is a file
		return []string{dir}, []string{endDir}
	}
	for _, file := range files {
		//add this directory to path names
		filePath := dir + "/" + file.Name()
		endPath := endDir + "/" + file.Name()
		//recurse to next level
		newPaths, newEndPaths := getDirFiles(filePath, endPath)
		filePaths = append(filePaths, newPaths...)
		endPaths = append(endPaths, newEndPaths...)
		//add endings of path names
	}
	return filePaths, endPaths
}

//ConfigTemplate takes a modifier object, a file path where the template is located, the target path, and two maps of data to populate the template with.
//It configures the template and writes it to the specified file path.
func ConfigTemplate(modifier *kv.Modifier, emptyFilePath string, configuredFilePath string, dataPaths ...string) {
	emptyTemplate, err := ioutil.ReadFile(emptyFilePath)
	template := string(emptyTemplate)
	if err != nil {
		panic(err)
	}
	//populate template
	template = PopulateTemplate(template, modifier, dataPaths...)
	popTemplate := []byte(template)
	//create new file
	newFile, err := os.Create(configuredFilePath)
	if err != nil {
		panic(err)
	}
	//write to file
	_, err = newFile.Write(popTemplate)
	if err != nil {
		panic(err)
	}
	newFile.Close()
}