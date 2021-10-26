package utils

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func GetStringInBetween(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str[s:], end)
	if e == -1 {
		return
	}
	return str[s : s+e]
}

func LineByLineDiff(stringA *string, stringB *string) string {
	//Colors used for output
	var Reset = "\033[0m"
	var Red = "\033[31m"
	var Green = "\033[32m"
	var Cyan = "\033[36m"
	var result string

	if runtime.GOOS == "windows" {
		Reset = "\x1b[0m"
		Red = "\x1b[31m"
		Green = "\x1b[32m"
		Cyan = "\x1b[36m"
	}

	dmp := diffmatchpatch.New()
	var patchText string
	var patchOutput string
	//Patch Calculation - Catches patch slice out of bounds
	func() {
		defer func() {
			if r := recover(); r != nil {
				patchText = ""
			}
		}()
		patches := dmp.PatchMake(*stringA, *stringB) //This throws out of index slice error rarely
		patchText = dmp.PatchToText(patches)
	}()

	if patchText != "" {
		//Converts escaped chars in patches
		unescapedPatchText, err2 := url.PathUnescape(patchText)
		if err2 != nil {
			log.Fatalf("Unable to decode percent-encoding: %v", err2)
		}

		parsedPatchText := strings.Split(unescapedPatchText, "\n")

		//Fixes char offset due to common preString
		for i, string := range parsedPatchText {
			if strings.Contains(string, "@@") {
				charOffset := string[strings.Index(parsedPatchText[i], "-")+1 : strings.Index(parsedPatchText[i], ",")]
				charOffsetInt, _ := strconv.Atoi(charOffset)
				charOffsetInt = charOffsetInt - 2 + len(parsedPatchText[i+1])
				parsedPatchText[i] = strings.Replace(string, charOffset, strconv.Itoa(charOffsetInt), 2)
			}
		}

		//Grabs only patch data from PatchMake
		onlyPatchedText := []string{}
		for _, stringLine := range parsedPatchText {
			if strings.Contains(stringLine, "@@") {
				onlyPatchedText = append(onlyPatchedText, stringLine)
			}
		}

		//Patch Data Output
		patchOutput = Cyan + strings.Join(onlyPatchedText, " ") + Reset + "\n"
	} else {
		patchOutput = Cyan + "@@ Patch Data Unavailable @@" + Reset + "\n"
	}

	//Diff Calculation
	timeOut := time.Date(9999, 1, 1, 12, 0, 0, 0, time.UTC)
	diffs := dmp.DiffBisect(*stringA, *stringB, timeOut)
	diffs = dmp.DiffCleanupSemantic(diffs)

	//Seperates diff into red and green lines
	var redBuffer bytes.Buffer
	var greenBuffer bytes.Buffer
	for _, diff := range diffs {
		text := diff.Text
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			_, _ = greenBuffer.WriteString(Green)
			_, _ = greenBuffer.WriteString(text)
			_, _ = greenBuffer.WriteString(Reset)
		case diffmatchpatch.DiffInsert:
			_, _ = redBuffer.WriteString(Red)
			_, _ = redBuffer.WriteString(text)
			_, _ = redBuffer.WriteString(Reset)
		case diffmatchpatch.DiffEqual:
			_, _ = redBuffer.WriteString(text)
			_, _ = greenBuffer.WriteString(text)
		}
	}

	greenLineSplit := strings.Split(greenBuffer.String(), "\n")
	redLineSplit := strings.Split(redBuffer.String(), "\n")

	//Adds + for each green line
	for greenIndex, greenLine := range greenLineSplit {
		if strings.Contains(greenLine, Green) {
			greenLineSplit[greenIndex] = "+" + greenLine
		}
	}

	//Adds - for each red line
	for redIndex, redLine := range redLineSplit {
		if strings.Contains(redLine, Red) {
			redLineSplit[redIndex] = "-" + redLine
		}
	}

	//Red vs Green length
	lengthDiff := 0
	sameLength := 0
	var redSwitch bool
	if len(redLineSplit) > len(greenLineSplit) {
		redSwitch = true
		lengthDiff = len(redLineSplit) - len(greenLineSplit)
		sameLength = len(greenLineSplit)
	} else { //Green > Red
		redSwitch = false
		lengthDiff = len(greenLineSplit) - len(redLineSplit)
		sameLength = len(redLineSplit)
	}

	//Prints line-by-line until shorter length
	currentIndex := 0
	for currentIndex != sameLength {
		redLine := redLineSplit[currentIndex]
		greenLine := greenLineSplit[currentIndex]
		if len(redLine) > 0 && redLine[0] == '-' {
			result += redLine + "\n"
		}
		if len(greenLine) > 0 && greenLine[0] == '+' {
			result += greenLine + "\n"
		}
		currentIndex++
	}

	//Prints rest of longer length
	for currentIndex != lengthDiff+sameLength {
		if redSwitch {
			redLine := redLineSplit[currentIndex]
			if len(redLine) > 0 && redLine[0] == '-' {
				result += redLine + "\n"
			}
		} else {
			greenLine := greenLineSplit[currentIndex]
			if len(greenLine) > 0 && greenLine[0] == '+' {
				result += greenLine + "\n"
			}
		}
		currentIndex++
	}

	//Colors first line "+" & "-"
	if len(result) > 0 && string(result[0]) == "+" {
		result = strings.Replace(result, "+", Green+"+"+Reset, 1)
	} else if len(result) > 0 && string(result[0]) == "-" {
		result = strings.Replace(result, "-", Red+"-"+Reset, 1)
	}

	//Colors all "+" & "-" using previous newline
	result = strings.ReplaceAll(result, "\n", Reset+"\n")
	result = strings.ReplaceAll(result, "\n+", "\n"+Green+"+"+Reset)
	result = strings.ReplaceAll(result, "\n-", "\n"+Red+"-"+Reset)

	//Diff vs no Diff output
	if len(strings.TrimSpace(result)) == 0 {
		if runtime.GOOS == "windows" {
			return "@@ No Differences @@"
		}
		return Cyan + "@@ No Differences @@" + Reset
	} else {
		result = patchOutput + result
		result = strings.TrimSuffix(result, "\n")
	}

	if runtime.GOOS == "windows" {
		result = strings.ReplaceAll(result, Reset, "")
		result = strings.ReplaceAll(result, Green, "")
		result = strings.ReplaceAll(result, Cyan, "")
		result = strings.ReplaceAll(result, Red, "")
	}

	return result
}

func VersionHelper(versionData map[string]interface{}, templateOrValues bool, valuePath string) {
	Reset := "\033[0m"
	Cyan := "\033[36m"
	Red := "\033[31m"
	if runtime.GOOS == "windows" {
		Reset = ""
		Cyan = ""
		Red = ""
	}

	if versionData == nil {
		fmt.Println("No version data found for this environment")
		os.Exit(1)
	}

	//template == true
	if templateOrValues {
		for _, versionMap := range versionData {
			for _, versionMetadata := range versionMap.(map[string]interface{}) {
				for field, data := range versionMetadata.(map[string]interface{}) {
					if field == "destroyed" && !data.(bool) {
						goto printOutput1
					}
				}
			}
		}
		return

	printOutput1:
		for filename, versionMap := range versionData {
			fmt.Println(Cyan + "======================================================================================")
			fmt.Println(filename)
			fmt.Println("======================================================================================" + Reset)
			keys := make([]int, 0, len(versionMap.(map[string]interface{})))
			for versionNumber := range versionMap.(map[string]interface{}) {
				versionNo, err := strconv.Atoi(versionNumber)
				if err != nil {
					fmt.Println()
				}
				keys = append(keys, versionNo)
			}
			sort.Ints(keys)
			for i, key := range keys {
				versionNumber := fmt.Sprint(key)
				versionMetadata := versionMap.(map[string]interface{})[fmt.Sprint(key)]
				fmt.Println("Version " + string(versionNumber) + " Metadata:")

				fields := make([]string, 0, len(versionMetadata.(map[string]interface{})))
				for field := range versionMetadata.(map[string]interface{}) {
					fields = append(fields, field)
				}
				sort.Strings(fields)
				for _, field := range fields {
					fmt.Printf(field + ": ")
					fmt.Println(versionMetadata.(map[string]interface{})[field])
				}
				if i != len(keys)-1 {
					fmt.Println(Red + "-------------------------------------------------------------------------------" + Reset)
				}
			}
		}
		fmt.Println(Cyan + "======================================================================================" + Reset)
	} else {
		for _, versionMetadata := range versionData {
			for field, data := range versionMetadata.(map[string]interface{}) {
				if field == "destroyed" && !data.(bool) {
					goto printOutput
				}
			}
		}
		return

	printOutput:
		fmt.Println(Cyan + "======================================================================================" + Reset)
		keys := make([]int, 0, len(versionData))
		for versionNumber := range versionData {
			versionNo, _ := strconv.ParseInt(versionNumber, 10, 64)
			keys = append(keys, int(versionNo))
		}
		sort.Ints(keys)
		for _, key := range keys {
			versionNumber := key
			versionMetadata := versionData[fmt.Sprint(key)]
			fields := make([]string, 0)
			fieldData := make(map[string]interface{}, 0)
			for field, data := range versionMetadata.(map[string]interface{}) {
				fields = append(fields, field)
				fieldData[field] = data
			}
			sort.Strings(fields)
			fmt.Println("Version " + fmt.Sprint(versionNumber) + " Metadata:")
			for _, field := range fields {
				fmt.Printf(field + ": ")
				fmt.Println(fieldData[field])
			}
			if keys[len(keys)-1] != versionNumber {
				fmt.Println(Red + "-------------------------------------------------------------------------------" + Reset)
			}
		}
		fmt.Println(Cyan + "======================================================================================" + Reset)
	}
}

func RemoveDuplicateValues(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func DiffHelper(resultMap map[string]*string, envLength int, envDiffSlice []string, fileSysIndex int, config bool, mutex *sync.Mutex) {
	fileIndex := 0
	keys := []string{}
	mutex.Lock()
	fileList := make([]string, len(resultMap)/envLength)
	mutex.Unlock()

	if config {
		//Make fileList
		for key := range resultMap {
			found := false
			keySplit := strings.Split(key, "||")

			for _, fileName := range fileList {
				if fileName == keySplit[1] {
					found = true
				}
			}

			if !found && len(fileList) > 0 {
				fileList[fileIndex] = keySplit[1]
				fileIndex++
			}
		}
	} else {
		for _, env := range envDiffSlice { //Arranges keys for ordered output
			if strings.Contains(env, "_0") {
				env = strings.Split(env, "_")[0]
			}
			keys = append(keys, env+"||"+env+"_seed.yml")
		}
		fileList[0] = "placeHolder"
	}

	//Diff resultMap using fileList
	for _, fileName := range fileList {
		if config {
			//Arranges keys for ordered output
			for _, env := range envDiffSlice {
				keys = append(keys, env+"||"+fileName)
			}
			if fileSysIndex == len(envDiffSlice) {
				keys = append(keys, "filesys||"+fileName)
			}
		}

		Reset := "\033[0m"
		Red := "\033[31m"
		Green := "\033[32m"
		Yellow := "\033[0;33m"

		if runtime.GOOS == "windows" {
			Reset = ""
			Red = ""
			Green = ""
			Yellow = ""
		}

		keyA := keys[0]
		keyB := keys[1]
		keySplitA := strings.Split(keyA, "||")
		keySplitB := strings.Split(keyB, "||")
		mutex.Lock()

		sortedKeyA := keyA
		sortedKeyB := keyB
		if _, ok := resultMap[sortedKeyA]; !ok {
			sortedKeyA = "||" + keySplitA[1]
		}
		if _, ok := resultMap[sortedKeyB]; !ok {
			sortedKeyB = "||" + keySplitA[1]
		}

		envFileKeyA := resultMap[sortedKeyA]
		envFileKeyB := resultMap[sortedKeyB]
		mutex.Unlock()

		latestVersionACheck := strings.Split(keySplitA[0], "_")
		if len(latestVersionACheck) > 1 && latestVersionACheck[1] == "0" {
			keySplitA[0] = strings.ReplaceAll(keySplitA[0], "0", "latest")
		}
		latestVersionBCheck := strings.Split(keySplitB[0], "_")
		if len(latestVersionBCheck) > 1 && latestVersionBCheck[1] == "0" {
			keySplitB[0] = strings.ReplaceAll(keySplitB[0], "0", "latest")
		}
		switch envLength {
		case 4:
			keyC := keys[2]
			keyD := keys[3]
			keySplitC := strings.Split(keyC, "||")
			keySplitD := strings.Split(keyD, "||")
			mutex.Lock()
			envFileKeyC := resultMap[keyC]
			envFileKeyD := resultMap[keyD]
			mutex.Unlock()

			latestVersionCCheck := strings.Split(keySplitC[0], "_")
			if len(latestVersionCCheck) > 1 && latestVersionCCheck[1] == "0" {
				keySplitC[0] = strings.ReplaceAll(keySplitC[0], "0", "latest")
			}
			latestVersionDCheck := strings.Split(keySplitD[0], "_")
			if len(latestVersionDCheck) > 1 && latestVersionDCheck[1] == "0" {
				keySplitD[0] = strings.ReplaceAll(keySplitD[0], "0", "latest")
			}

			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitA[0] + Reset + Green + " +Env-" + keySplitB[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyB, envFileKeyA))
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitA[0] + Reset + Green + " +Env-" + keySplitC[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyC, envFileKeyA))
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitA[0] + Reset + Green + " +Env-" + keySplitD[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyD, envFileKeyA))
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitB[0] + Reset + Green + " +Env-" + keySplitC[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyC, envFileKeyB))
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitB[0] + Reset + Green + " +Env-" + keySplitD[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyD, envFileKeyB))
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitC[0] + Reset + Green + " +Env-" + keySplitD[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyD, envFileKeyC))
		case 3:
			keyC := keys[2]
			keySplitC := strings.Split(keyC, "||")
			mutex.Lock()
			envFileKeyC := resultMap[keyC]
			mutex.Unlock()

			latestVersionCCheck := strings.Split(keySplitC[0], "_")
			if len(latestVersionCCheck) > 1 && latestVersionCCheck[1] == "0" {
				keySplitC[0] = strings.ReplaceAll(keySplitC[0], "0", "latest")
			}

			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitA[0] + Reset + Green + " +Env-" + keySplitB[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyB, envFileKeyA))
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitA[0] + Reset + Green + " +Env-" + keySplitC[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyC, envFileKeyA))
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitB[0] + Reset + Green + " +Env-" + keySplitC[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyC, envFileKeyB))
		default:
			fmt.Print("\n" + Yellow + keySplitA[1] + " (" + Reset + Red + "-Env-" + keySplitA[0] + Reset + Green + " +Env-" + keySplitB[0] + Reset + Yellow + ")" + Reset + "\n")
			fmt.Println(LineByLineDiff(envFileKeyB, envFileKeyA))
		}

		//Seperator
		if runtime.GOOS == "windows" {
			fmt.Printf("======================================================================================\n")
		} else {
			fmt.Printf("\033[1;35m======================================================================================\033[0m\n")
		}
		keys = keys[:0] //Cleans keys for next file
	}
}
