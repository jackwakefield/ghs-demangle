package main

import "os"
import "fmt"
import "bufio"
import "strings"
import "unicode"
import "unicode/utf8"
import "errors"
import "regexp"
import "strconv"

var templatePrefixes = []string{"tm", "ps", "pt"}

var baseNames = map[string]string{
	"__vtbl": " virtual table",
	"__ct":   "#",
	"__dt":   "~#",
	"__as":   "operator=",
	"__eq":   "operator==",
	"__ne":   "operator!=",
	"__gt":   "operator>",
	"__lt":   "operator<",
	"__ge":   "operator>=",
	"__le":   "operator<=",
	"__pp":   "operator++",
	"__pl":   "operator+",
	"__apl":  "operator+=",
	"__mi":   "operator-",
	"__ami":  "operator-=",
	"__ml":   "operator*",
	"__amu":  "operator*=",
	"__dv":   "operator/",
	/* XXX below baseNames have not been seen - guess from libiberty cplus-dem.c*/
	"__adv": "operator/=",
	"__nw":  "operator new",
	"__dl":  "operator delete",
	"__vn":  "operator new[]",
	"__vd":  "operator delete[]",
	"__md":  "operator%",
	"__amd": "operator%=",
	"__mm":  "operator--",
	"__aa":  "operator&&",
	"__oo":  "operator||",
	"__or":  "operator|",
	"__aor": "operator|=",
	"__er":  "operator^",
	"__aer": "operator^=",
	"__ad":  "operator&",
	"__aad": "operator&=",
	"__co":  "operator~",
	"__cl":  "operator",
	"__ls":  "operator<<",
	"__als": "operator<<=",
	"__rs":  "operator>>",
	"__ars": "operator>>=",
	"__rf":  "operator->",
	"__vc":  "operator[]",
}

var baseTypes = map[rune]string{
	'v': "void",
	'i': "int",
	's': "short",
	'c': "char",
	'w': "wchar_t",
	'b': "bool",
	'f': "float",
	'd': "double",
	'l': "long",
	'L': "long long",
	'e': "...",
	/* XXX below baseTypes have not been seen - guess from libiberty cplus-dem.c */
	'r': "long double",
}

var typePrefixes = map[rune]string{
	'U': "unsigned",
	'S': "signed",
	/* XXX below typePrefixes have not been seen - guess from libiberty cplus-dem.c */
	'J': "__complex",
}

var typeSuffixes = map[rune]string{
	'P': "*",
	'R': "&",
	'C': "const",
	'V': "volatile", /* XXX this is a guess! */
	/* XXX below typeSuffixes have not been seen - guess from libiberty cplus-dem.c */
	'u': "restrict",
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var filename = os.Args[1]

	fmt.Println("Hello", filename)

	demangleAll(filename)
	fmt.Println("")
	fmt.Println(extractInt("14SoftSimGatewayFf"))
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
				fmt.Println("Error demangling:", line)
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
	if strings.HasPrefix(mangle, "S__") {
		declStatic = "static "
		mangle = mangle[3:]
	}

	var declNameSpace = ""
	var declClass = ""
	if strings.HasPrefix(mangle, "Q") {
		declNameSpace, mangle, err = readNameSpace(mangle)
		check(err)

		var last = strings.LastIndex(declNameSpace, "::")
		if last == -1 {
			declClass = declNameSpace
		} else {
			declClass = declNameSpace[last+2:]
		}

		declNameSpace += "::"
	} else if startsWithDigit(mangle) {
		declClass, mangle, err = readString(mangle)
		check(err)
		declNameSpace = declClass + "::"
	}

	baseName = strings.Replace(baseName, "#", declClass, -1)

	if strings.HasPrefix(mangle, "S") {
		declStatic = "static "
		mangle = mangle[1:]
	}

	var declConst = ""
	if strings.HasPrefix(mangle, "C") {
		declConst = " const"
		mangle = mangle[1:]
	}

	var declType = "#"
	if strings.HasPrefix(mangle, "F") {
		var args = []string{}
		declType, mangle, err = readType(args, mangle)
		check(err)
	}

	/*
		var end
		if strings.HasPrefix(mangle, "_") {
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

	return strings.Replace(declStatic+partTwo+declConst, "::"+baseNames["__vtbl"], baseNames["__vtbl"], -1), nil
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
	var mstart = strings.Index(name, "__")
	if mstart != -1 && strings.HasPrefix(name[mstart:], "___") {
		mstart++
	}

	if mstart == -1 {
		return name, nil
	}

	//var remainder = name[mstart+2 : ]
	name = name[0:mstart]

	return name, nil
}

func readBaseName(name string) (string, string, error) {
	if len(name) == 0 {
		return "", "", errors.New("Unexpected end of string, Expected a name.")
	}

	var opName = ""
	if strings.HasPrefix(name, "__op") {
		var args = []string{}
		var t, name, err = readType(args, name)

		check(err)
		opName = "operator " + t
		name = "#" + name
	}

	var mstart = strings.Index(name, "__")
	if mstart != -1 && strings.HasPrefix(name[mstart:], "___") {
		mstart++
	}

	if mstart == -1 {
		var remainder = ""
		return name, remainder, nil
	}

	var remainder = name[mstart+2:]
	name = name[:mstart]

	if val, ok := baseNames[name]; ok {
		name = val
	} else if name == "#" {
		name = opName
	}

	for startsWithAny(remainder, templatePrefixes) {
		var lstart = strings.Index(remainder, "__")
		if lstart == -1 {
			return "", "", errors.New("Bad template argument")
		}

		name += "__" + remainder[:lstart]
		remainder = remainder[lstart+2:]

		var remainder, length, err = extractInt(remainder)
		if err != nil || length == 0 || length > len(remainder) {
			return "", "", errors.New("Bad template argument length.")
		}

		name += "__" + strconv.Itoa(length) + remainder
		if len(remainder) == 0 {
			return name, remainder, nil
		}

		if !strings.HasPrefix(remainder, "__") {
			return "", "", errors.New("Unexpected character(s) after template.")
		}

		remainder = remainder[2:]
	}

	var dt, err = demangleTemplate(name)
	check(err)

	return dt, remainder, nil
}

func readArguments(name string) (string, string, error) {
	var remainder = name
	return "", remainder, nil
}

func readTemplateArguments(name string) (string, string, error) {
	var remainder = name
	return "", remainder, nil
}

func readType(args []string, name string) (string, string, error) {
	var remainder = name
	return "", remainder, nil
}

func readNameSpace(name string) (string, string, error) {
	var remainder = name
	return "", remainder, nil
}

func readString(input string) (string, string, error) {
	if len(input) == 0 {
		return "", "", errors.New("Unexpected end of string.  Expected a digit.")
	}

	var name, length, err = extractInt(input)
	check(err)
	if length == 0 || len(name) < length {
		return "", "", errors.New("Bad string length.")
	}

	var remainder = input[length:]
	var dt = ""
	dt, err = demangleTemplate(name)
	check(err)

	return dt, remainder, nil
}

func extractInt(name string) (string, int, error) {
	if len(name) == 0 {
		return "", 0, errors.New("Unexpected end of string.  Expected a digit.")
	}
	re := regexp.MustCompile(`([0-9]+)(.*)$`)
	var results = re.FindStringSubmatch(name)
	//Should have 3 items in array; the whole regex match, and then each capture.
	if results == nil || len(results) < 3 {
		return "", 0, errors.New("Unexpected end of string.  Unable to match digits.")
	}

	var len, err = strconv.Atoi(results[1])
	check(err)
	return results[2], len, nil
}

func startsWithAny(input string, names []string) bool {
	for _, v := range names {
		if strings.HasPrefix(input, v) {
			return true
		}
	}
	return false
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
