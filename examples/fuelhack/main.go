package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"time"

	simconnect "github.com/bmurray/simconnect-go"
	"github.com/bmurray/simconnect-go/client"
)

var programLevel = new(slog.LevelVar)

func main() {
	gals := flag.Int("gals", 20, "The number of gallons the tank should contain after fueling")
	minFuel := flag.Float64("min", 1.0, "The minimum fuel before adding more")
	debug := flag.Bool("debug", false, "debug")
	flag.Parse()

	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))

	if *debug {
		programLevel.Set(slog.LevelDebug)
	}

	slog.Debug("Adding Fuel", "gals", *gals)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// Configure our application fuel limits
	ff := &refuel{
		left:    float64(*gals),
		right:   float64(*gals),
		minFuel: *minFuel,
	}
	con := simconnect.NewConnector("refuel", simconnect.WithReceiver(ff))
	con.StartReconnect(ctx)
}

// FuelRequest is the data structure to set fuel levels in the main tanks
type FuelRequest struct {
	client.RecvSimobjectDataByType
	FuelLevelLeftMain  float64 `name:"FUEL TANK LEFT MAIN QUANTITY" unit:"Gallons"`
	FuelLevelRightMain float64 `name:"FUEL TANK RIGHT MAIN QUANTITY" unit:"Gallons"`
}

// FuelReport is the data structure to report fuel levels inall tanks
type FuelReport struct {
	client.RecvSimobjectDataByType
	FuelLevelCenter    float64 `name:"FUEL TANK CENTER QUANTITY" unit:"Gallons"`
	FuelLevelCenter2   float64 `name:"FUEL TANK CENTER2 QUANTITY" unit:"Gallons"`
	FuelLevelCenter3   float64 `name:"FUEL TANK CENTER3 QUANTITY" unit:"Gallons"`
	FuelLevelExternal1 float64 `name:"FUEL TANK EXTERNAL1 QUANTITY" unit:"Gallons"`
	FuelLevelExternal2 float64 `name:"FUEL TANK EXTERNAL2 QUANTITY" unit:"Gallons"`
	FuelLevelLeftMain  float64 `name:"FUEL TANK LEFT MAIN QUANTITY" unit:"Gallons"`
	FuelLevelLeftAux   float64 `name:"FUEL TANK LEFT AUX QUANTITY" unit:"Gallons"`
	FuelLevelLeftTip   float64 `name:"FUEL TANK LEFT TIP QUANTITY" unit:"Gallons"`
	FuelLevelRightMain float64 `name:"FUEL TANK RIGHT MAIN QUANTITY" unit:"Gallons"`
	FuelLevelRightAux  float64 `name:"FUEL TANK RIGHT AUX QUANTITY" unit:"Gallons"`
	FuelLevelRightTip  float64 `name:"FUEL TANK RIGHT TIP QUANTITY" unit:"Gallons"`
}

type refuel struct {
	left    float64
	right   float64
	minFuel float64
}

// Start is called when the refuel receiver is started
// it gets called after the connection is established
// and whenever a reconnection happens
func (r *refuel) Start(ctx context.Context, sc *client.SimConnect) {
	slog.Debug("Starting refuel")

	// You MUST register the data definitions before you can request or set data
	// The most convenient way to do this is to register them in the Start method
	// as the start method is called after the connection is established
	// and whenever a reconnection happens
	if err := sc.RegisterDataDefinition(&FuelReport{}); err != nil {
		slog.Error("Cannot register report", "error", err)
		return
	}
	if err := sc.RegisterDataDefinition(&FuelRequest{}); err != nil {
		slog.Error("Cannot register report", "error", err)
		return
	}

	// Start a goroutine to request fuel levels every 5 seconds
	// This ensures we always have the latest fuel levels
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Stopping refuel")
				return
			case <-time.After(5 * time.Second):
				// fr := &FuelReport{}
				if err := simconnect.RequestData[FuelReport](sc); err != nil {
					slog.Error("Cannot request fuel", "error", err)
					continue
				}

			}
		}
	}(ctx)
}

// Update is called whenever a new data packet is received
func (r *refuel) Update(ctx context.Context, sc *client.SimConnect, ppData *client.RecvSimobjectDataByType) {

	// Ensure the data is data we want
	if fr, is := simconnect.IsReport[FuelReport](sc, ppData); is {

		// Print the data; this is just for us, and not required
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(fr); err != nil {
			slog.Error("Error encoding report", "error", err)
		}

		// If the fuel levels are above the minimum, we are done
		if fr.FuelLevelLeftMain >= r.minFuel && fr.FuelLevelRightMain >= r.minFuel {
			slog.Debug("Fuel level OK")
			return
		}

		// If we have no fuel to add, we are done
		// eg, if someone passes -gals 0, we don't want to add fuel
		if r.left == 0 && r.right == 0 {
			slog.Debug("No fuel to add")
			return
		}

		// Create a new FuelRequest with the fuel levels we want to set
		tq := &FuelRequest{
			FuelLevelLeftMain:  r.left,
			FuelLevelRightMain: r.right,
		}
		// err := tq.SetData(sc)
		err := sc.SetData(tq)
		if err != nil {
			slog.Error("Cannot set fuel", "error", err)
		}
	}
}
