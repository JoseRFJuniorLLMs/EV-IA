package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

var (
	serverURL   = flag.String("server", "ws://localhost:9000/ocpp", "OCPP server WebSocket URL")
	chargePointID = flag.String("id", "CP001", "Charge Point ID")
	vendor      = flag.String("vendor", "SIGEC", "Charge Point Vendor")
	model       = flag.String("model", "SimulatorV1", "Charge Point Model")
	serial      = flag.String("serial", "SIM001", "Serial Number")
	firmware    = flag.String("firmware", "1.0.0", "Firmware Version")
	v2gCapable  = flag.Bool("v2g", false, "Enable V2G capability")
	batterySOC  = flag.Int("soc", 80, "Battery State of Charge (%) for V2G")
	batteryCapacity = flag.Float64("battery", 75.0, "Battery capacity (kWh) for V2G")
	maxChargePower = flag.Float64("charge-power", 150.0, "Max charge power (kW)")
	maxDischargePower = flag.Float64("discharge-power", 50.0, "Max V2G discharge power (kW)")
	connectorCount = flag.Int("connectors", 2, "Number of connectors")
	interactive = flag.Bool("interactive", false, "Enable interactive mode")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Setup logger
	var logger *zap.Logger
	var err error
	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create simulator config
	config := &SimulatorConfig{
		ServerURL:         *serverURL,
		ChargePointID:     *chargePointID,
		Vendor:            *vendor,
		Model:             *model,
		SerialNumber:      *serial,
		FirmwareVersion:   *firmware,
		V2GCapable:        *v2gCapable,
		BatterySOC:        *batterySOC,
		BatteryCapacityKWh: *batteryCapacity,
		MaxChargePowerKW:  *maxChargePower,
		MaxDischargePowerKW: *maxDischargePower,
		ConnectorCount:    *connectorCount,
	}

	// Create and start simulator
	simulator := NewSimulator(config, logger)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down simulator...")
		simulator.Stop()
		os.Exit(0)
	}()

	// Connect to server
	if err := simulator.Connect(); err != nil {
		logger.Fatal("Failed to connect to server", zap.Error(err))
	}

	// Start the simulator
	if *interactive {
		runInteractiveMode(simulator, logger)
	} else {
		// Run in background mode
		fmt.Printf("OCPP Charge Point Simulator started\n")
		fmt.Printf("  ID: %s\n", *chargePointID)
		fmt.Printf("  Server: %s\n", *serverURL)
		fmt.Printf("  V2G: %v\n", *v2gCapable)
		fmt.Println("\nPress Ctrl+C to stop")

		// Keep running
		select {}
	}
}

func runInteractiveMode(sim *Simulator, logger *zap.Logger) {
	fmt.Println("\nOCPP Charge Point Simulator - Interactive Mode")
	fmt.Println("============================================")
	fmt.Println("Commands:")
	fmt.Println("  start <connector>       - Start charging on connector")
	fmt.Println("  stop                    - Stop current charging")
	fmt.Println("  status <connector>      - Set connector status (Available/Occupied/Faulted)")
	fmt.Println("  meter <value>           - Send meter value (Wh)")
	fmt.Println("  heartbeat               - Send heartbeat")
	fmt.Println("  v2g start <power>       - Start V2G discharge (kW)")
	fmt.Println("  v2g stop                - Stop V2G discharge")
	fmt.Println("  v2g soc <percent>       - Set battery SOC")
	fmt.Println("  fault <connector>       - Simulate fault on connector")
	fmt.Println("  reset                   - Simulate device reset")
	fmt.Println("  firmware accept|reject  - Respond to firmware update")
	fmt.Println("  quit                    - Exit simulator")
	fmt.Println("")

	sim.RunInteractive()
}
