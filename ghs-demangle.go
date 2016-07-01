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

var baseTypes = map[byte]string{
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

var typePrefixes = map[byte]string{
	'U': "unsigned",
	'S': "signed",
	/* XXX below typePrefixes have not been seen - guess from libiberty cplus-dem.c */
	'J': "__complex",
}

var typeSuffixes = map[byte]string{
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
	//fmt.Println("")
	//fmt.Println(demangle("__ct__7MyClass"))
}

func demangleAll(filename string) {
	file, err := os.Open(filename)
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line = scanner.Text()
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
		declType, mangle, err = readType(nil, mangle)
		check(err)
	}

	if strings.HasPrefix(mangle, "_") {
		var end, err = strconv.Atoi(mangle[1:])
		check(err)
		baseName += "_" + strconv.Itoa(end)
		mangle = ""
	}

	if len(mangle) > 0 {
		return "", fmt.Errorf("Unknown modifier %v", mangle)
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
	var mstart = strings.Index(name[1:], "__")
	if mstart != -1 && strings.HasPrefix(name[mstart:], "___") {
		mstart++
	}

	if mstart == -1 {
		return name, nil
	}

	var remainder = name[mstart+3:]
	name = name[:mstart+1]

	for true {
		if !startsWithAny(remainder, templatePrefixes) {
			return "", errors.New("Unexpected template argument prefix.")
		}

		var lstart = strings.Index(remainder, "__")
		if lstart == -1 {
			return "", errors.New("Bad template argument")
		}

		remainder = remainder[lstart+2:]

		var name, remainder, err = extractName(remainder)
		if err != nil {
			return "", errors.New("Bad template argument length.")
		}

		if !strings.HasPrefix(name, "_") { //This checks 'name', instead of 'remainder' because of the refactoring of extractName
			return "", fmt.Errorf("Unexpected character after template parameter length. \"%v\"", name)
		}

		var declArgs = ""
		var tmp = ""
		declArgs, tmp, err = readTemplateArguments(name[1:])
		check(err)

		if strings.HasSuffix(declArgs, ">") {
			declArgs += " "
		}

		name += "<" + declArgs + ">"

		if tmp != remainder {
			return "", fmt.Errorf("Bad template argument length %v != %v", tmp, remainder)
		}

		if len(remainder) == 0 {
			return name, nil
		}

		if !strings.HasPrefix(remainder, "__") {
			return "", errors.New("Unexpected character(s) after template.")
		}

		remainder = remainder[2:]
	}

	return name, nil
}

func readBaseName(name string) (string, string, error) {
	if len(name) == 0 {
		return "", "", errors.New("Unexpected end of string, Expected a name.")
	}

	var opName = ""
	if strings.HasPrefix(name, "__op") {
		var t, name, err = readType(nil, name)

		check(err)
		opName = "operator " + t
		name = "#" + name
	}

	var mstart = strings.Index(name[1:], "__")
	if mstart != -1 && strings.HasPrefix(name[mstart:], "___") {
		mstart++
	}

	if mstart == -1 {
		var remainder = ""
		return name, remainder, nil
	}

	var remainder = name[mstart+3:]
	name = name[:mstart+1]

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

		var name, remainder, err = extractName(remainder)
		if err != nil {
			return "", "", errors.New("Bad template argument length.")
		}

		name += "__" + strconv.Itoa(len(name)) + remainder
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
	var result = ""
	var args = []string{}
	var remainder = name
	var err error = nil

	for len(remainder) > 0 && !strings.HasPrefix(remainder, "_") {
		if len(args) > 0 {
			result += ", "
		}

		var t = ""
		t, remainder, err = readType(args, remainder)
		check(err)

		result += strings.Replace(t, "#", "", -1)
		args = append(args, t)
	}

	return result, remainder, nil
}

func readTemplateArguments(name string) (string, string, error) {
	var result = ""
	var args = []string{}
	var remainder = name

	var tipe = "" //Can't call it 'type' as the original source does :)
	var val = ""
	for len(remainder) > 0 && !strings.HasPrefix(remainder, "_") {
		var err error = nil
		if len(args) > 0 {
			result += ", "
		}

		if strings.HasPrefix(remainder, "X") {
			remainder = remainder[1:]
			if len(remainder) == 0 {
				return "", "", errors.New("Unexpected end of string.  Expected a type.")
			}

			if startsWithDigit(remainder) {
				tipe = "#"
				val, remainder, err = readString(remainder)
				check(err)
			} else {
				tipe, remainder, err = readType(args, remainder)
				check(err)
				tipe = strings.Replace(tipe, "#", " #", -1)
				if strings.HasPrefix(remainder, "L") {
					remainder = remainder[1:]
					if len(remainder) == 0 || remainder[0] != '_' {
						return "", "", errors.New("Unexpected end of string.  Expected '_'.")
					}

					val, remainder, err = extractName(remainder[1:])

					if err != nil {
						return "", "", errors.New("Bad template parameter length.")
					}

					if !strings.HasPrefix(val, "_") { //This checks 'name', instead of 'remainder' because of the refactoring of extractName
						return "", "", fmt.Errorf("Unexpected character after template parameter length. \"%v\"", val)
					}
				}
			}
		} else {
			val, remainder, err = readType(args, remainder)
			check(err)
			tipe = "class #"
		}

		result += strings.Replace(tipe, "#", "Z"+strconv.Itoa(len(args)+1)+" = "+val, -1)
		args = append(args, val)
	}

	return result, remainder, nil
}

func readType(args []string, name string) (string, string, error) {
	fmt.Println("readType", args, name)
	if len(name) == 0 {
		return "", "", errors.New("Unexpected end of string.  Expected a type.")
	}

	if val, ok := baseTypes[name[0]]; ok {
		return val + "#", name[1:], nil
	} else if strings.HasPrefix(name, "Q") {
		var result, remainder, err = readNameSpace(name)
		check(err)
		return result + "#", remainder, nil
	} else if startsWithDigit(name) {
		var result, remainder, err = readString(name)
		check(err)
		return result + "#", remainder, nil
	} else if val, ok := typePrefixes[name[0]]; ok {
		var result, remainder, err = readType(args, name[1:])
		check(err)
		return val + " " + result, remainder, nil
	} else if val, ok := typeSuffixes[name[0]]; ok {
		var result, remainder, err = readType(args, name[1:])
		check(err)
		return strings.Replace(result, "#", " "+val+"#", -1), remainder, nil
	} else if strings.HasPrefix(name, "Z") {
		var index = strings.Index(name[1:], "Z") //next 'Z'
		if index == -1 {
			return "", "", errors.New("Unexpected end of string. Expected \"Z\"")
		}
		return name[:index] + "#", name[:index+1], nil
	} else if strings.HasPrefix(name, "A") {
		name = name[1:]
		/*
			var index = strings.Index(name, "_Z")
			if index > -1 {
				var end = strings.Index(name[2:], "Z") //next 'Z'
				length = strconv.Atoi(name[1 : end-1])
				name = name[end+1:]
			} else {
		*/
		var length, remainder, err = readIntPrefix(name)

		if err != nil || len(remainder) == 0 {
			return "", "", fmt.Errorf("Unexpected end of string.  Expected \"_\".")
		}
		if !strings.HasPrefix(remainder, "_") {
			return "", "", fmt.Errorf("Unexpected character after array length \"%c\".  Expected \"_\".", remainder[0])
		}
		remainder = remainder[1:] //skip over '_'
		var result = ""
		result, remainder, err = readType(args, remainder)
		check(err)
		return strings.Replace(result, "#", "#["+strconv.Itoa(length)+"]", -1), remainder, nil
	} else if strings.HasPrefix(name, "F") {
		var declArgs, name, err = readArguments(name[1:])
		check(err)
		if args == nil && (len(name) == 0 || strings.HasPrefix(name, "_")) {
			return "#(" + declArgs + ")", name, nil
		}
		if len(name) == 0 {
			return "", "", errors.New("Unexpected end of string, expected \"_\".")
		}
		if !strings.HasPrefix(name, "_") {
			return "", "", fmt.Errorf("Unexpected character after template parameter length. \"%v\"", name)
		}
		var result, remainder = "", ""
		result, remainder, err = readType(args, name[1:])
		check(err)
		return strings.Replace(result, "#", "(#)("+declArgs+")", -1), remainder, nil
	} else if strings.HasPrefix(name, "T") {
		var index, name, err = readIntPrefix(name[1:])
		check(err)
		if len(args) < index {
			return "", "", fmt.Errorf("Bad argument number \"%v\".", index)
		}

		return args[index-1], name, nil
	} else if strings.HasPrefix(name, "N") {

	}

	return "", "", fmt.Errorf("Unknown type \"%c\".", name[0])
}

func readNameSpace(input string) (string, string, error) {
	if len(input) == 0 || input[0] != 'Q' {
		return "", "", errors.New("Unexpected end of string.  Expected \"Q\".")
	}

	var namespaces = []string{}

	var count, remainder, err = readIntPrefix(input[1:])
	check(err)

	remainder = remainder[1:] //step over '_'

	for i := 0; i < count; i++ {
		//var remainderspaces = strings.SplitAfter(remainder, "Z")
		var ns = ""
		ns, remainder, err = readString(remainder)
		check(err)
		namespaces = append(namespaces, ns)
	}

	fmt.Println("readNameSpace:", input, "=>", namespaces, remainder)
	return strings.Join(namespaces, "::"), remainder, nil
}

func readString(input string) (string, string, error) {
	if len(input) == 0 {
		return "", "", errors.New("Unexpected end of string.  Expected a digit.")
	}

	var name, remainder, err = extractName(input)
	check(err)

	var dt = ""
	dt, err = demangleTemplate(name)
	check(err)

	return dt, remainder, nil
}

func readIntPrefix(input string) (int, string, error) {
	if len(input) == 0 {
		return -1, "", errors.New("Unexpected end of string.  Expected a digit.")
	}

	re := regexp.MustCompile(`([0-9]+)(.*)$`)
	var results = re.FindStringSubmatch(input)

	//Should have 3 items in array; the whole regex match, and then each capture.
	if results == nil || len(results) < 3 {
		return -1, "", errors.New("Unexpected end of string.  Unable to match digits.")
	}

	var length, err = strconv.Atoi(results[1])
	check(err)
	var postNumber = results[2]
	//fmt.Println("readIntPrefix", input, length, postNumber)
	return length, postNumber, nil
}

func extractName(name string) (string, string, error) {
	var length, postNumber, err = readIntPrefix(name)
	return postNumber[:length], postNumber[length:], err
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
		//fmt.Println(e)
	}
}
