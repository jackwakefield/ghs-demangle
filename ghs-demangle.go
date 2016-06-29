package main

import "os"
import "fmt"
import "bufio"
import "strings"
import "unicode"
import "unicode/utf8"
import "errors"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var filename = os.Args[1]

	fmt.Println("Hello", filename)
	demangleAll(filename)
}

func demangleAll(filename string) {
	file, err := os.Open(filename)
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line = scanner.Text()
		//fmt.Println(scanner.Text())
		if len(line) > 0 {
			var name, err = demangle(line)
			if err != nil {
				fmt.Println("Error demangling: ", line)
			}
			fmt.Println(name)
		}
	}
}

func demangle(input string) (string, error) {
	var name, err = decompress(input)
	check(err)

	var baseName = ""
	var mangle = ""
	baseName, mangle, err = readBaseName(name)
	check(err)

	var declStatic = ""
	if strings.HasPrefix("S__", mangle) {
		declStatic = "static "
		mangle = mangle[3:len(mangle)]
	}

	var declNameSpace = ""
	var declClass = ""
	if strings.HasPrefix("Q", mangle) {
		declNameSpace, mangle, err = readNameSpace(mangle)
		check(err)

		var last = strings.LastIndex("::", declNameSpace)
		if last == -1 {
			declClass = declNameSpace
		} else {
			declClass = declNameSpace[last+2 : len(declNameSpace)]
		}

		declNameSpace += "::"
	} else if startsWithDigit(mangle) {
		declClass, mangle, err = readString(mangle)
		check(err)
		declNameSpace = declClass + "::"
	}

	baseName = strings.Replace(baseName, "#", declClass, -1)

	if strings.HasPrefix("S", mangle) {
		declStatic = "static "
		mangle = mangle[1:len(mangle)]
	}

	var declConst = ""
	if strings.HasPrefix("C", mangle) {
		declConst = " const"
		mangle = mangle[1:len(mangle)]
	}

	var declType = "#"
	if strings.HasPrefix("F", mangle) {
		var args = []string{}
		declType, mangle, err = readType(args, mangle)
		check(err)
	}

	/*
		var end
		if strings.HasPrefix("_", mangle) {
			baseName += "_" + end.ToString
			mangle = ""
		}
	*/

	if len(mangle) > 0 {
		fmt.Println("Unknown modifier: " + mangle)
		return "", errors.New("Unknown modifier")
	}

	var partTwo = declType
	partTwo = strings.Replace(partTwo, "(#)", " "+declNameSpace+baseName, -1)
	partTwo = strings.Replace(partTwo, "#", declNameSpace+baseName, -1)

	return strings.Replace(declStatic+partTwo+declConst, ":: virtual table", " virtual table", -1), nil
}

func startsWithDigit(input string) bool {
	if len(input) == 0 {
		return false
	}
	r, size := utf8.DecodeRuneInString(input[0:1])
	if size > 0 {
		return unicode.IsDigit(r)
	}
	return false
}

func decompress(name string) (string, error) {
	return name, nil
}

func demangleTemplate(name string) (string, error) {
	return name, nil
}

func readBaseName(name string) (string, string, error) {
	if len(name) == 0 {
		return "", "", errors.New("Unexpected end of string, Expected a name.")
	}

	var remainder = ""
	return name, remainder, nil
}

func readArguments(name string) (string, string, error) {
	var remainder = ""
	return name, remainder, nil
}

func readTemplateArguments(name string) (string, string, error) {
	var remainder = ""
	return name, remainder, nil
}

func readType(args []string, name string) (string, string, error) {
	var remainder = ""
	return name, remainder, nil
}

func readNameSpace(name string) (string, string, error) {
	var remainder = ""
	return name, remainder, nil
}

func readString(name string) (string, string, error) {
	var remainder = ""
	return name, remainder, nil
}

func printUsage() {
	fmt.Println("Ported to Golang from Chadderz121/ghs-demangle")
	fmt.Println("ghs-demangle [FILE]")
	fmt.Println("Demangles all symbols in FILE to standard output.")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
