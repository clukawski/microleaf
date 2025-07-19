package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/user"
	"strconv"

	"github.com/spf13/viper"
)

const defaultConfigFile = ".microleafrc"

var configFilePath string
var panelName string
var verbose = flag.Bool("v", false, "Verbose")
var config *MicroleafConfig

// HostConfig defines the structure for individual host configurations.
type HostConfig struct {
	PanelName   string `mapstructure:"panel_name,required"`
	Host        string `mapstructure:"host,required"`
	AccessToken string `mapstructure:"access_token,required"`
}

// MicroleafConfig defines the overall structure of the configuration file.
type MicroleafConfig struct {
	HostConfigs []HostConfig `mapstructure:"host_configs"`
}

func initConfig() {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("error: failed to look up current user: %v\n", err)
		os.Exit(1)
	}
	defaultConfigFilePath := usr.HomeDir
	flag.StringVar(&configFilePath, "f", defaultConfigFilePath, "Config file path")
	flag.StringVar(&panelName, "n", "", "Panel name")
	flag.Parse()

	// Ensure the user has provided a panel name to search
	// the config for.
	if panelName == "" {
		usage()
	}

	// Initialize Viper
	v := viper.New()

	// Set the config file name without extension
	v.SetConfigName(defaultConfigFile)
	// Set the config file type
	v.SetConfigType("toml")

	// Set the path where Viper should look for the config file
	v.AddConfigPath(configFilePath)
	v.AddConfigPath(defaultConfigFilePath)

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("error: failed to read in config file: %v\n", err)
	}

	// Unmarshal the config into the MicroleafConfig struct
	var c MicroleafConfig
	if err := v.Unmarshal(&c); err != nil {
		log.Fatalf("error: failed to parse config file: %v\n", err)
	}
	config = &c
}

func usage() {
	fmt.Println("usage: microleaf -n <panel_name> [-f <path>] [-v] <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println()
	fmt.Println("   on           Turn on Nanoleaf")
	fmt.Println("   off          Turn off Nanoleaf")
	fmt.Println()
	fmt.Println("   effect       Control Nanoleaf effects")
	fmt.Println("   panel        Control Nanoleaf panel")
	fmt.Println()
	fmt.Println("   hsl          Set Nanoleaf to the provided HSL")
	fmt.Println("   rgb          Set Nanoleaf to the provided RGB")
	fmt.Println("   temp         Set Nanoleaf to the provided color temperature")
	fmt.Println("   brightness   Set Nanoleaf to the provided brightness")
	fmt.Println()
	fmt.Println("   get          Send a GET request to the Nanoleaf")
	fmt.Println()
	os.Exit(1)
}

func main() {
	initConfig()

	if *verbose {
		fmt.Printf("configs: %+v\n\n", config.HostConfigs)
	}

	var client *Client
	for n, hostConfig := range config.HostConfigs {
		if hostConfig.PanelName == panelName {
			client = &Client{
				Host:    hostConfig.Host,
				Token:   hostConfig.AccessToken,
				Verbose: *verbose,
			}
			if *verbose {
				fmt.Printf(
					"current config [%d]: %s\n\n",
					n, hostConfig,
				)
			}
			break
		}
	}
	if client == nil {
		log.Println("error: no config matching specified panel name")
		usage()
	}

	if flag.NArg() > 0 {
		cmd := flag.Arg(0)
		switch cmd {
		case "brightness":
			doBrightnessCommand(client, flag.Args()[1:])
		case "effect":
			doEffectCommand(client, flag.Args()[1:])
		case "get":
			doGetCommand(client, flag.Args()[1:])
		case "hsl":
			doHSLCommand(client, flag.Args()[1:])
		case "off":
			err := client.Off()
			if err != nil {
				fmt.Println("error: failed to turn off Nanoleaf:", err)
				os.Exit(1)
			}
		case "on":
			err := client.On()
			if err != nil {
				fmt.Println("error: failed to turn on Nanoleaf:", err)
				os.Exit(1)
			}
		case "panel":
			doPanelCommand(client, flag.Args()[1:])
		case "rgb":
			doRGBCommand(client, flag.Args()[1:])
		case "temp":
			doColorTemperatureCommand(client, flag.Args()[1:])
		default:
			usage()
		}
	} else {
		usage()
	}
}

func doBrightnessCommand(client *Client, args []string) {
	if len(args) < 1 {
		fmt.Println("usage: microleaf brightness <brightness>")
		os.Exit(1)
	}

	brightness, err := strconv.Atoi(args[0])
	if err != nil || brightness < 0 || brightness > 100 {
		fmt.Println("error: temperature must be an integer 0-100")
		os.Exit(1)
	}

	err = client.SetBrightness(brightness)
	if err != nil {
		fmt.Println("error: failed to set brightness:", err)
		os.Exit(1)
	}
}

func doColorTemperatureCommand(client *Client, args []string) {
	if len(args) < 1 {
		fmt.Println("usage: microleaf temp <temperature>")
		os.Exit(1)
	}

	temp, err := strconv.Atoi(args[0])
	if err != nil || temp < 1200 || temp > 6500 {
		fmt.Println("error: temperature must be an integer 1200-6500")
		os.Exit(1)
	}

	err = client.SetColorTemperature(temp)
	if err != nil {
		fmt.Println("error: failed to set color temperature:", err)
		os.Exit(1)
	}
}

func doEffectCommand(client *Client, args []string) {
	usage := func() {
		fmt.Println("usage: microleaf effect list")
		fmt.Println("       microleaf effect select <name>")
		fmt.Println("       microleaf effect custom [<panel> <red> <green> <blue> <transition time>] ...")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	command := args[0]
	switch command {
	case "custom":
		customArgs := args[1:]
		numFrameArgs := 5
		if len(customArgs)%numFrameArgs != 0 {
			fmt.Println("usage: microleaf effect custom [<panel> <red> <green> <blue> <transition time>] ...")
		}

		numFrames := len(customArgs) / numFrameArgs
		frames := make([]SetPanelColor, numFrames)
		for i := 0; i < numFrames; i++ {
			offset := numFrameArgs * i
			panelID, err := strconv.ParseUint(customArgs[offset], 10, 16)
			if err != nil {
				fmt.Printf("error: expected panel ID between 0-%d, got %s", math.MaxUint16, customArgs[offset])
				os.Exit(1)
			}

			red, err := strconv.ParseUint(customArgs[offset+1], 10, 8)
			if err != nil {
				fmt.Printf("error: expected red value between 0-%d, got %s", math.MaxUint8, customArgs[offset+1])
				os.Exit(1)
			}

			green, err := strconv.ParseUint(customArgs[offset+2], 10, 8)
			if err != nil {
				fmt.Printf("error: expected green value between 0-%d, got %s", math.MaxUint8, customArgs[offset+2])
				os.Exit(1)
			}

			blue, err := strconv.ParseUint(customArgs[offset+3], 10, 8)
			if err != nil {
				fmt.Printf("error: expected blue value between 0-%d, got %s", math.MaxUint8, customArgs[offset+3])
				os.Exit(1)
			}

			transitionTime, err := strconv.ParseUint(customArgs[offset+4], 10, 16)
			if err != nil {
				fmt.Printf("error: expected transition time between 0-%d, got %s", math.MaxUint16, customArgs[offset+4])
				os.Exit(1)
			}

			frames[i].PanelID = uint16(panelID)
			frames[i].Red = uint8(red)
			frames[i].Green = uint8(green)
			frames[i].Blue = uint8(blue)
			frames[i].TransitionTime = uint16(transitionTime)
		}

		err := client.SetCustomColors(frames)
		if err != nil {
			fmt.Println("error: failed to start external control:", err)
			os.Exit(1)
		}
	case "list":
		list, err := client.ListEffects()
		if err != nil {
			fmt.Println("error: failed retrieve effects list:", err)
			os.Exit(1)
		}
		for _, name := range list {
			fmt.Println(name)
		}
	case "select":
		if len(args) != 2 {
			fmt.Println("usage: microleaf effect select <name>")
			os.Exit(1)
		}

		name := args[1]
		err := client.SelectEffect(name)
		if err != nil {
			fmt.Println("error: failed to select effect:", err)
			os.Exit(1)
		}
	default:
		usage()
	}
}

func doGetCommand(client *Client, args []string) {
	if len(args) < 1 {
		fmt.Println("usage: microleaf get <path>")
		os.Exit(1)
	}

	res, err := client.Get(args[0])
	if err != nil {
		fmt.Println("error: failed to set color temperature:", err)
		os.Exit(1)
	}

	fmt.Println(res)
}

func doPanelCommand(client *Client, args []string) {
	usage := func() {
		fmt.Println("usage: microleaf panel info")
		fmt.Println("       microleaf panel model")
		fmt.Println("       microleaf panel name")
		fmt.Println("       microleaf panel version")
		os.Exit(1)
	}

	if len(args) != 1 {
		usage()
	}

	panelInfo, err := client.GetPanelInfo()
	if err != nil {
		fmt.Println("error: failed to get Nanoleaf state:", err)
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "info":
		fmt.Println("Name:", panelInfo.Name)
		fmt.Println()
		fmt.Println("Manufacturer:", panelInfo.Manufacturer)
		fmt.Println("Model:       ", panelInfo.Model)
		fmt.Println("Serial No:   ", panelInfo.SerialNo)
		fmt.Println()
		fmt.Println("Firmware Version:", panelInfo.FirmwareVersion)
		fmt.Println()
		fmt.Println("State:")
		fmt.Println("  On:  ", panelInfo.State.On.Value)
		fmt.Println("  Mode:", panelInfo.State.ColorMode)
		fmt.Println()
		fmt.Printf("  Hue:        %3d° [%d°-%d°]\n", panelInfo.State.Hue.Value, *panelInfo.State.Hue.Min, *panelInfo.State.Hue.Max)
		fmt.Printf("  Saturation: %3d  [%d-%d]\n", panelInfo.State.Saturation.Value, *panelInfo.State.Saturation.Min, *panelInfo.State.Saturation.Max)
		fmt.Printf("  Brightness: %3d  [%d-%d]\n", panelInfo.State.Brightness.Value, *panelInfo.State.Brightness.Min, *panelInfo.State.Brightness.Max)
		fmt.Println()
		fmt.Printf("  Color Temperature: %4dK [%dK-%dK]\n", panelInfo.State.ColorTemperature.Value, *panelInfo.State.ColorTemperature.Min, *panelInfo.State.ColorTemperature.Max)
		fmt.Println()
		fmt.Println("Effects:")
		fmt.Println("  Selected:", panelInfo.Effects.Selected)
		fmt.Println("  Available:")
		for _, effect := range panelInfo.Effects.List {
			fmt.Println("  -", effect)
		}
		fmt.Println()
		fmt.Println("Layout:")
		fmt.Printf("  Orientation: %d° [%d°-%d°]\n", panelInfo.PanelLayout.GlobalOrientation.Value, panelInfo.PanelLayout.GlobalOrientation.Min, panelInfo.PanelLayout.GlobalOrientation.Max)
		fmt.Println("  Panels:     ", panelInfo.PanelLayout.Layout.NumPanels)
		fmt.Println("  Side Length:", panelInfo.PanelLayout.Layout.SideLength)
		fmt.Println()
		fmt.Println("  Panel Positions:")
		for _, panel := range panelInfo.PanelLayout.Layout.PositionData {
			fmt.Printf("  - %3d: (%d, %d, %d°)\n", panel.PanelID, panel.X, panel.Y, panel.O)
		}
		fmt.Println()
		fmt.Println("Rhythm:")
		fmt.Println("  ID:      ", panelInfo.Rhythm.ID)
		fmt.Printf("  Position: (%.0f, %.0f, %.0f°)\n", panelInfo.Rhythm.Position.X, panelInfo.Rhythm.Position.Y, panelInfo.Rhythm.Position.O)
		fmt.Println()
		fmt.Println("  Connected:    ", panelInfo.Rhythm.Connected)
		fmt.Println("  Aux Available:", panelInfo.Rhythm.AuxAvailable)
		fmt.Println("  Active:       ", panelInfo.Rhythm.Active)
		fmt.Println("  Mode:         ", panelInfo.Rhythm.Mode)
		fmt.Println()
		fmt.Println("  Versions:")
		fmt.Println("    Hardware:", panelInfo.Rhythm.HardwareVersion)
		fmt.Println("    Firmware:", panelInfo.Rhythm.FirmwareVersion)
		fmt.Println()
	case "layout":
		fmt.Printf("Orientation: %d° [%d°-%d°]\n", panelInfo.PanelLayout.GlobalOrientation.Value, panelInfo.PanelLayout.GlobalOrientation.Min, panelInfo.PanelLayout.GlobalOrientation.Max)
		fmt.Println("Panels:     ", panelInfo.PanelLayout.Layout.NumPanels)
		fmt.Println("Side Length:", panelInfo.PanelLayout.Layout.SideLength)
		fmt.Println()
		fmt.Println("Positions:")
		for _, panel := range panelInfo.PanelLayout.Layout.PositionData {
			fmt.Printf("- %3d: (%d, %d, %d°)\n", panel.PanelID, panel.X, panel.Y, panel.O)
		}
		fmt.Println()
	case "model":
		fmt.Println(panelInfo.Model)
	case "name":
		fmt.Println(panelInfo.Name)
	case "state":
		fmt.Println("On:  ", panelInfo.State.On.Value)
		fmt.Println("Mode:", panelInfo.State.ColorMode)
		fmt.Println()
		fmt.Printf("Brightness: %3d [%d-%d]\n", panelInfo.State.Brightness.Value, *panelInfo.State.Brightness.Min, *panelInfo.State.Brightness.Max)
		fmt.Printf("Hue:        %3d [%d-%d]\n", panelInfo.State.Hue.Value, *panelInfo.State.Hue.Min, *panelInfo.State.Hue.Max)
		fmt.Printf("Saturation: %3d [%d-%d]\n", panelInfo.State.Saturation.Value, *panelInfo.State.Saturation.Min, *panelInfo.State.Saturation.Max)
		fmt.Println()
		fmt.Printf("Color Temperature: %4dK [%dK-%dK]\n", panelInfo.State.ColorTemperature.Value, *panelInfo.State.ColorTemperature.Min, *panelInfo.State.ColorTemperature.Max)
		fmt.Println()
	case "version":
		fmt.Println("Panel Firmware:", panelInfo.FirmwareVersion)
		fmt.Println()
		fmt.Println("Rhythm:")
		fmt.Println("  Hardware:", panelInfo.Rhythm.HardwareVersion)
		fmt.Println("  Firmware:", panelInfo.Rhythm.FirmwareVersion)
		fmt.Println()
	default:
		usage()
	}
}

func doHSLCommand(client *Client, args []string) {
	if len(args) != 3 {
		fmt.Println("usage: microleaf hsl <hue> <saturation> <lightness>")
		os.Exit(1)
	}

	hue, err := strconv.Atoi(args[0])
	if err != nil || hue < 0 || hue > 360 {
		fmt.Println("error: hue must be an integer 0-100")
		os.Exit(1)
	}

	sat, err := strconv.Atoi(args[1])
	if err != nil || sat < 0 || sat > 100 {
		fmt.Println("error: saturation must be an integer 0-360")
		os.Exit(1)
	}

	lightness, err := strconv.Atoi(args[2])
	if err != nil || lightness < 0 || lightness > 100 {
		fmt.Println("error: lightness must be an integer 0-100")
		os.Exit(1)
	}

	err = client.SetHSL(hue, sat, lightness)
	if err != nil {
		fmt.Println("error: failed to set HSL:", err)
		os.Exit(1)
	}
}

func doRGBCommand(client *Client, args []string) {
	if len(args) != 3 {
		fmt.Println("usage: microleaf rgb <red> <green> <blue>")
		os.Exit(1)
	}

	red, err := strconv.Atoi(args[0])
	if err != nil || red < 0 || red > 255 {
		fmt.Println("error: red must be an integer 0-255")
		os.Exit(1)
	}

	green, err := strconv.Atoi(args[1])
	if err != nil || green < 0 || green > 255 {
		fmt.Println("error: green must be an integer 0-255")
		os.Exit(1)
	}

	blue, err := strconv.Atoi(args[2])
	if err != nil || blue < 0 || blue > 255 {
		fmt.Println("error: blue must be an integer 0-255")
		os.Exit(1)
	}

	err = client.SetRGB(red, green, blue)
	if err != nil {
		fmt.Println("error: failed to set RGB:", err)
		os.Exit(1)
	}
}
