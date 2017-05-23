package main

import (
	"os"
	"flag"
	"strings"
	"path/filepath"
	"github.com/fatih/color"
	"text/template"
)

func main() {
	color.New(color.FgBlue, color.Bold).Println("Welcome to Gonfigen, the Gonfig generator")

	tn := flag.String("type", "", "Targetted type")
	pckg := flag.String("package", "", "Targetted package")
	pr := flag.String("root", "..", "Project root")
	flag.Parse()

	typeNames := *tn
	packg := *pckg
	projectRoot := *pr

	if typeNames == "" {
		panic("No type set")
	}

	if packg == "" {
		color.Blue("No package set, guessing from current directory.\n")
		gopath := os.Getenv("GOPATH")
		if (gopath == "") {
			panic("Can not guess package with no $GOPATH set!")
		}

		currDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			panic(err)
		}

		gopathParts := strings.Split(gopath, string(os.PathListSeparator))
		for _, p := range(gopathParts) {
			srcDir := p + string(os.PathSeparator) + "src" + string(os.PathSeparator)

			if strings.HasPrefix(currDir, srcDir) {
				packg = strings.TrimPrefix(currDir, srcDir)
				break
			}
		}

		if (packg == "") {
			panic("Can not guess package from available $GOPATH!")
		}
	}

	color.New(color.FgBlue).Printf("Working with %s package and %s type.\n", packg, typeNames)

	currDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	cmdDir := currDir + string(os.PathSeparator) + projectRoot + string(os.PathSeparator) + "cmd" + string(os.PathSeparator) + "gonfig" + string(os.PathSeparator)
	os.MkdirAll(cmdDir, 0777)

	f, err := os.Create(cmdDir + string(os.PathSeparator) + "gonfig.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	packageParts := strings.Split(packg, "/")
	typeNames = packageParts[len(packageParts)-1] + "." + typeNames

	asd := struct{
		Package string
		Type string
	}{
		Package: packg,
		Type: typeNames,
	}

	var tpl = template.Must(template.New("").Parse(tplData))
	if err = tpl.Execute(f, asd); err != nil {
		panic(err)
	}

	pth, err := filepath.Abs(f.Name())
	if err != nil {
		panic(err)
	}

	color.New(color.FgGreen, color.Bold).Printf("Gonfig CLI generated to %s\n", pth)

	f2, err := os.Create(currDir + string(os.PathSeparator) + projectRoot + string(os.PathSeparator) + "gonfig_loaders.go")
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	var tpl2 = template.Must(template.New("").Parse(GonfigLoadersTplData))
	if err = tpl2.Execute(f2, asd); err != nil {
		panic(err)
	}

	pth2, err := filepath.Abs(f2.Name())
	if err != nil {
		panic(err)
	}

	color.New(color.FgGreen, color.Bold).Printf("Gonfig loaders generated to %s\n", pth2)
}

const tplData = `package main

import (
	"reflect"
	"bufio"
	"os"
	"github.com/fatih/color"
	"github.com/BurntSushi/toml"
	"flag"
	"strings"
	"strconv"
	"fmt"
	"{{.Package}}"
)

func main() {
	destination := *flag.String("destination", "gonfig.toml", "Path generated toml file")
	template := *flag.String("template", "", "Path to template toml file")
	nonInteractive := flag.Bool("non-interactive", false, "Dump the template/defaults")
	flag.Parse()

	regeneratingDestination := false
	if template == "" {
		template = "gonfig.dist.toml"
		if _, err := os.Stat(destination); err == nil {
			template = destination
			regeneratingDestination = true
		}
	}

	color.New(color.FgBlue, color.Bold).Println("Welcome to Gonfig!")

	var n = {{.Type}}{}
	color.New(color.FgBlue).Printf("Working with %s configuration object.\n", reflect.TypeOf(n))

	if regeneratingDestination {
		color.New(color.FgBlue).Printf("Destination (%s) already exists, loading values from it.\n", template)
	} else {
		color.New(color.FgBlue).Printf("Reading template data from %s\n", template)
	}
	toml.DecodeFile(template, &n)

	if *nonInteractive {
		color.Blue("Non-interactive mode: skipping user input.")
	} else {
		color.Blue("Now let's setup the configuration values:")
		setStructValue("", &n)
		color.Green("Thank you, that will be all. Generating config...")
	}

	// open output file
	fo, err := os.Create(destination)
	if err != nil {
		panic(err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	toml.NewEncoder(fo).Encode(n);

	color.New(color.FgGreen, color.Bold).Printf("Config successfully generated to '%s'. Exiting.", destination)
}

func setStructValue(name string, s interface{}) {
	numField := reflect.TypeOf(s).Elem().NumField()
	rv := reflect.ValueOf(s).Elem()
	for i := 0; i < numField; i++ {
		// if struct, recursion?
		field := rv.Field(i)
		n := rv.Type().Field(i).Name
		if name != "" {
			n = fmt.Sprintf(" %s", n)
		}

		setFieldValue(n, field, 0)
	}
}

func setFieldValue(name string, field reflect.Value, indent int) (error) {
	//todo: move this out at some point
	reader := bufio.NewReader(os.Stdin)
	p := printer{indent}

	FieldInputLoop:
	for ; ; {
		switch field.Kind() {
		case reflect.String:
			s := color.New(color.FgYellow, color.Bold).Sprint(name) + color.New(color.FgYellow).Sprintf(" (%s)", field) + color.New(color.FgYellow, color.Bold).Sprint(": ")
			p.Print(s)

			line, tooLong, err := reader.ReadLine()
			if tooLong {
				p.Print(color.RedString("Input too long"))
			} else if err != nil {
				p.Print(color.RedString("Unable to parse! %s", err))
			} else {
				if len(line) > 0 {
					field.SetString(string(line))
				}

				break FieldInputLoop
			}
		case reflect.Bool:
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(name))
			if field.Bool() {
				p.Print(color.New(color.FgYellow).Sprintf(" (Y/n): ", ))
			} else {
				p.Print(color.New(color.FgYellow).Sprintf(" (y/N): ", ))
			}

			line, tooLong, err := reader.ReadLine()
			if tooLong {
				p.Print(color.RedString("Input too long."))
			} else if err != nil {
				p.Print(color.RedString("Unable to parse. %s", err))
			} else if l := strings.ToLower(string(line)); l != "y" && l != "n" && l != "" {
				p.Print(color.RedString("Invalid value, only Y/y/N/n allowed."))
			} else {
				if len(line) > 0 {
					field.SetBool(strings.ToLower(string(line)) == "y")
				}

				break FieldInputLoop
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(name))
			p.Print(color.New(color.FgYellow).Sprintf(" (%v)", field.Int()))
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(": "))

			line, tooLong, err := reader.ReadLine()
			if tooLong {
				p.Print(color.RedString("Input too long."))
			} else if err != nil {
				p.Print(color.RedString("Unable to parse. %s", err))
			} else {
				if string(line) == "" {
					break FieldInputLoop
				}

				bitSize := field.Type().Bits()
				val, err := strconv.ParseInt(string(line), 10, bitSize)
				if err != nil {
					message := "Error parsing integer."

					if e, ok := err.(*strconv.NumError); ok {
						switch e.Err {
						case strconv.ErrSyntax:
							message += " Wrong syntax."
						case strconv.ErrRange:
							message += fmt.Sprintf(" Value out of range, destination type is %s.", field.Kind())
						}
					}

					p.Print(color.RedString(message))
				} else {
					field.Set(reflect.ValueOf(val).Convert(field.Type()))

					break FieldInputLoop
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(name))
			p.Print(color.New(color.FgYellow).Sprintf(" (%v)", field.Uint()))
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(": "))

			line, tooLong, err := reader.ReadLine()
			if tooLong {
				p.Print(color.RedString("Input too long."))
			} else if err != nil {
				p.Print(color.RedString("Unable to parse. %s", err))
			} else {
				if string(line) == "" {
					break FieldInputLoop
				}

				bitSize := field.Type().Bits()
				val, err := strconv.ParseUint(string(line), 10, bitSize)
				if err != nil {
					message := "Error parsing unsigned integer."

					if e, ok := err.(*strconv.NumError); ok {
						switch e.Err {
						case strconv.ErrSyntax:
							message += " Wrong syntax."
						case strconv.ErrRange:
							message += fmt.Sprintf(" Value out of range, destination type is %s.", field.Kind())
						}
					}

					p.Print(color.RedString(message))
				} else {
					field.Set(reflect.ValueOf(val).Convert(field.Type()))

					break FieldInputLoop
				}
			}
		case reflect.Float32, reflect.Float64:
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(name))
			p.Print(color.New(color.FgYellow).Sprintf(" (%v)", field.Float()))
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(": "))

			line, tooLong, err := reader.ReadLine()
			if tooLong {
				p.Print(color.RedString("Input too long."))
			} else if err != nil {
				p.Print(color.RedString("Unable to parse. %s", err))
			} else {
				if string(line) == "" {
					break FieldInputLoop
				}

				bitSize := field.Type().Bits()
				val, err := strconv.ParseFloat(string(line), bitSize)
				if err != nil {
					message := "Error parsing float."

					if e, ok := err.(*strconv.NumError); ok {
						switch e.Err {
						case strconv.ErrSyntax:
							message += " Wrong syntax."
						case strconv.ErrRange:
							message += fmt.Sprintf(" Value out of range, destination type is %s.", field.Kind())
						}
					}

					p.Print(color.RedString(message))
				} else {
					field.Set(reflect.ValueOf(val).Convert(field.Type()))

					break FieldInputLoop
				}
			}
		case reflect.Array, reflect.Slice:
			SliceLoop:
			for ; ; {

				//load/show defaults
				//remove/add items
				p.Print(color.New(color.FgYellow, color.Bold).Sprintln(name + ":"))
				if field.Len() > 0 {
					p.Print(color.New(color.FgYellow).Sprintf("  Existing values (index : value):\n"))

					for i := 0; i < field.Len(); i++ {
						p.Print(color.New(color.FgYellow).Sprintf("  %v: %v\n", i, field.Index(i)))
					}

					p.Print(color.New(color.FgYellow).Sprintf("  (A)dd/(R)emove/(P)urge/(N)ext (N): "))
				} else {
					p.Print(color.New(color.FgYellow).Sprintf("  (A)dd/(P)urge/(N)ext (N): "))
				}

				line, tooLong, err := reader.ReadLine()
				if tooLong {
					p.Print(color.RedString("  Input too long."))
				} else if err != nil {
					p.Print(color.RedString("  Unable to parse. %s", err))
				} else {
					if string(line) == "" {
						line = []byte("n")
					}

					switch strings.ToLower(string(line)) {
					case "n": break SliceLoop
					case "p":
						field.Set(reflect.New(field.Type()).Elem())
					case "r":
						p.Print(color.New(color.FgYellow).Sprintf("  Index to remove: "))

						line, tooLong, err := reader.ReadLine()
						if tooLong {
							p.Print(color.RedString("  Input too long."))
						} else if err != nil {
							p.Print(color.RedString("  Unable to parse. %s", err))
						} else {
							val, err := strconv.ParseInt(string(line), 10, strconv.IntSize)
							if err != nil {
								message := "  Error parsing integer."

								if e, ok := err.(*strconv.NumError); ok {
									switch e.Err {
									case strconv.ErrSyntax:
										message += " Wrong syntax."
									case strconv.ErrRange:
										message += " Value out of range, destination type is int64."
									}
								}

								p.Print(color.RedString(message))
							} else {
								if int(val) >= field.Len() {
									p.Print(color.RedString("  Index out of bounds. Index must be between 0 and %v", field.Len()))
								} else {
									field.Set(reflect.AppendSlice(field.Slice(0, int(val)), field.Slice(int(val) + 1, field.Len())))
								}
								//break FieldInputLoop
							}
						}
					case "a":
						x := reflect.New(field.Type().Elem()).Elem()
						setFieldValue(fmt.Sprintf("%s.%v", name, field.Len()), x, indent + 1)
						field.Set(reflect.Append(field, x))
					default:
						p.Print(color.RedString("  Unknown operation. Only A/a/R/r/P/p/N/n are allowed."))
					}

					p.Print(color.YellowString("  ---------------\n"))
				}
			}
			break FieldInputLoop
		case reflect.Map:
			MapLoop:
			for ; ; {

				//load/show defaults
				//remove/add items
				p.Print(color.New(color.FgYellow, color.Bold).Sprintln(name + ":"))
				if field.Len() > 0 {
					p.Print(color.New(color.FgYellow).Sprintf("  Existing values (index : value):\n"))

					keys := field.MapKeys()
					for i := 0; i < len(keys); i++ {
						p.Print(color.New(color.FgYellow).Sprintf("    %v : %v\n", keys[i], field.MapIndex(keys[i])))
					}

					p.Print(color.New(color.FgYellow).Sprintf("  (A)dd/(R)emove/(P)urge/(N)ext (N): "))
				} else {
					p.Print(color.New(color.FgYellow).Sprintf("  (A)dd/(P)urge/(N)ext (N): "))
				}

				line, tooLong, err := reader.ReadLine()
				if tooLong {
					p.Print(color.RedString("  Input too long."))
				} else if err != nil {
					p.Print(color.RedString("  Unable to parse. %s", err))
				} else {
					if string(line) == "" {
						line = []byte("n")
					}

					switch strings.ToLower(string(line)) {
					case "n": break MapLoop
					case "p":
						field.Set(reflect.New(field.Type()).Elem())
					case "r":
						x := reflect.New(field.Type().Key()).Elem()
						setFieldValue(fmt.Sprint("  Index to remove:"), x, indent + 1)

						if field.MapIndex(x) == (reflect.Value{}) {
							p.Print(color.RedString("  Key not found."))
						} else {
							field.SetMapIndex(x, reflect.Value{})
						}
					case "a":
						k := reflect.New(field.Type().Key()).Elem()
						setFieldValue(fmt.Sprint("  New index:"), k, indent + 1)
						v := reflect.New(field.Type().Elem()).Elem()
						setFieldValue(fmt.Sprint("  New value:"), v, indent + 1)

						if field.Len() == 0  {
							field.Set(reflect.MakeMap(reflect.MapOf(field.Type().Key(), field.Type().Elem())))
						}

						field.SetMapIndex(k, v)
					default:
						p.Print(color.RedString("  Unknown operation. Only A/a/R/r/P/p/N/n are allowed."))
					}

					p.Print(color.YellowString("---------------\n"))
				}
			}
			break FieldInputLoop
		case reflect.Struct:
			p.Print(color.New(color.FgYellow, color.Bold).Sprint(name + ":\n"))

			numField := field.NumField()
			for i := 0; i < numField; i++ {
				f := field.Field(i)
				n := field.Type().Field(i).Name
				if name != "" {
					n = fmt.Sprintf("%s", n)
				}

				setFieldValue(n, f, indent + 1)
			}

			break FieldInputLoop
		default: panic(field.Kind())
		}
	}

	return nil
}

type printer struct {
	indent int
}

func (p printer) Print(s string) {
	fmt.Print(strings.Repeat(" ", p.indent * 2) + s)
}

func (p printer) Println(s string) {
	fmt.Println(strings.Repeat(" ", p.indent * 2) + s)
}`

const GonfigLoadersTplData = `package main

import (
	"path/filepath"
	"os"
	"github.com/BurntSushi/toml"
	"{{.Package}}"
)

func LoadConfig() (n {{.Type}}, err error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return
	}

	_, err = toml.DecodeFile(dir + string(os.PathSeparator) + "gonfig.toml", &n)
	return
}
`