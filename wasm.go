package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"syscall/js"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.com/karakuritech/dk/kent/proto/kentpb"
)

const (
	NB_OF_DC_MOTORS        = 6
	NB_OF_PID_SETTINGS     = 4
	NB_OF_MASS_SETTINGS    = 4
	NB_OF_STEPPERS         = 12
	NB_OF_TEMP_CONTROLLERS = 2
	NB_OF_SCALES           = 5
	NB_OF_TRANSPORTS       = 11
	NB_OF_HANDOVER_POS     = 10
)

/*
Ctx - Context
*/
type Ctx struct {
	wsSrv  js.Value
	wsConn bool
}

type jsonData struct {
	ID     string `json:"id"`
	Binary string `json:"binary"`
}

type scaleData struct {
	Idx     string `json:"idx"`
	CalSamp string `json:"calSamp"`
	TarSamp string `json:"tarSamp"`
	RSamp   string `json:"rSamp"`
	CalW    string `json:"calW"`
	ZSamp   string `json:"zeroSamp"`
	TrayW   string `json:"trayW"`
}

type massData struct {
	Idx             string `json:"idx"`
	RunMx           string `json:"runMx"`
	DispenseTimeout string `json:"DispenseTimeoutMs"`
}

type pidData struct {
	Idx    string `json:"idx"`
	FKp    string `json:"fKp"`
	FKi    string `json:"fKi"`
	FKd    string `json:"fKd"`
	SatMx  string `json:"satMx"`
	SatMn  string `json:"satMn"`
	Dt     string `json:"dt"`
	Offset string `json:"offset"`
	SampT  string `json:"sampT"`
}

type stepperData struct {
	Idx         string `json:"idx"`
	Dir         string `json:"dir"`
	FSpeedMxRps string `json:"fSpeedMxRps"`
	SpeedPer    string `json:"speedPer"`
	FAccelRpss  string `json:"fAccelRpss"`
	CurrMx      string `json:"currMx"`
	CurrMn      string `json:"currMn"`
	HoldCurr    string `json:"holdCurr"`
	RetSpeedPer string `json:"retSpeedPer"`
	RetAng      string `json:"retAng"`
}

type dcMotorData struct {
	Idx         string `json:"idx"`
	Dir         string `json:"dir"`
	SpeedPer    string `json:"speedPer"`
	RetSpeedPer string `json:"retSpeedPer"`
	RetTimeMs   string `json:"retTimeMs"`
}

type eepromData struct {
	Scale    scaleData   `json:"scale"`
	Mass     massData    `json:"mass"`
	Pid      pidData     `json:"pid"`
	Stepper  stepperData `json:"stepper"`
	Dc_motor dcMotorData `json:"dc_motor"`
}

func (ctx *Ctx) getElementByID(elem string) js.Value {
	return js.Global().Get("document").Call("getElementById", elem)
}

func (ctx *Ctx) getElementString(elem string, value string) string {
	return ctx.getElementByID(elem).Get(value).String()
}

func (ctx *Ctx) getDispenserID() string {
	dispenserID := ctx.getElementString("txtDispenserId", "value")
	js.Global().Set("output", dispenserID)
	return dispenserID
}

func (ctx *Ctx) sendToWs(id string, data *kentpb.SrvToCli) {

	b, err := proto.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling", err)
	}

	var payload jsonData
	payload.ID = id
	payload.Binary = base64.StdEncoding.EncodeToString([]byte(b))

	p, _ := json.Marshal(payload)
	ctx.wsSrv.Call("send", string(p))
	ctx.appendToLog(data.String())

}

func (ctx *Ctx) receiveFromWs(msg js.Value) {

	//unmarshal to JSON
	var payload jsonData
	err := json.Unmarshal([]byte(msg.String()), &payload)
	if err != nil {
		fmt.Println("unmarshalling error. " + err.Error())
		return
	}

	//decode binary
	b, err := base64.StdEncoding.DecodeString(payload.Binary)

	//We're expecting only CliToSrv reports
	rpt := &kentpb.CliToSrv{}
	err = proto.Unmarshal(b, rpt)
	if err != nil {
		fmt.Println("Error unmarshaling", err)
		return
	}

	str := payload.ID + "\n" + rpt.String()

	if payload.ID == ctx.getDispenserID() {
		//append to correct log
		ctx.appendToLog(str)

		if rpt.GetDispenserPidDbgRpt() != nil {
			ctx.appendToPidLog(rpt)
		} else if rpt.GetEepromRRpt() != nil {
			ctx.parseEepromRead(rpt)
		}
	}
}

func (ctx *Ctx) appendToLog(msg string) {

	t := time.Now()

	timeStr := "[" + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute()) + ":" + strconv.Itoa(t.Second()) + "]"
	ctx.getElementByID("txtLogArea").Set("value", timeStr+" "+msg+"\n\n"+ctx.getElementString("txtLogArea", "value"))
}

func (ctx *Ctx) appendToPidLog(msg *kentpb.CliToSrv) {

	append := ""

	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetRun()+1)) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetCounter())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetTime())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetSp())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetCv())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetFError())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetFInteg())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetFDeriv())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetP())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetI())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetD())) + "\t"
	append += strconv.Itoa(int(msg.GetDispenserPidDbgRpt().GetPv())) + "\n"

	append = ctx.getElementByID("txtPidArea").Get("value").String() + append
	ctx.getElementByID("txtPidArea").Set("value", append)
}

func (ctx *Ctx) parseEepromRead(msg *kentpb.CliToSrv) {

	ctx.getElementByID("eepromExportData").Set("value", msg.String())

	factoryRpt := msg.GetEepromRRpt().GetFactoryRpt()
	if factoryRpt != nil {
		ctx.getElementByID("txtFactoryDispenserId").Set("value", factoryRpt.GetId())
		mac := factoryRpt.GetMac()
		ctx.getElementByID("txtFactoryMac").Set("value", fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]))
		ctx.getElementByID("cmbFactoryType").Set("value", int(factoryRpt.GetDeviceType()))
		ctx.getElementByID("txtFactoryHwRev").Set("value", factoryRpt.GetHwRev())
	}

	stepperRpt := msg.GetEepromRRpt().GetStepperRpt()
	if stepperRpt != nil {
		idx, _ := strconv.ParseUint(ctx.getElementString("txtStepperIdx", "value"), 10, 32)
		print("idx")
		println(idx)
		if idx >= NB_OF_STEPPERS {
			idx = NB_OF_STEPPERS - 1
		}
		ctx.getElementByID("txtStepperDir").Set("value", stepperRpt[idx].GetDirection())
		ctx.getElementByID("txtStepperSpeedRps").Set("value", float64(stepperRpt[idx].GetFSpeedMaxRps())/1000)
		ctx.getElementByID("txtStepperAccelRps").Set("value", float64(stepperRpt[idx].GetFAccelRpss())/1000)
		ctx.getElementByID("txtStepperDecelRps").Set("value", float64(stepperRpt[idx].GetFDecelRpss())/1000)
		ctx.getElementByID("txtStepperHomeSpeedRps").Set("value", float64(stepperRpt[idx].GetFHomeSpeedRps())/1000)
		ctx.getElementByID("txtStepperHomeAccelRps").Set("value", float64(stepperRpt[idx].GetFHomeAccelRpss())/1000)
		ctx.getElementByID("txtStepperMaxCurrent").Set("value", stepperRpt[idx].GetCurrentMax())
		ctx.getElementByID("txtStepperMinCurrent").Set("value", stepperRpt[idx].GetCurrentMin())
		ctx.getElementByID("txtStepperHoldCurrent").Set("value", stepperRpt[idx].GetHoldCurrent())
		ctx.getElementByID("txtStepperRetreatSpeed").Set("value", stepperRpt[idx].GetRetreatSpeedPct())
		ctx.getElementByID("txtStepperRetreatAngle").Set("value", stepperRpt[idx].GetRetreatAngle())
	}

	pidRpt := msg.GetEepromRRpt().GetPidRpt()
	if pidRpt != nil {
		idx, _ := strconv.ParseUint(ctx.getElementString("txtPidIdx", "value"), 10, 32)
		if idx >= NB_OF_PID_SETTINGS {
			idx = NB_OF_PID_SETTINGS - 1
		}
		ctx.getElementByID("txtDispenseKp").Set("value", float64(pidRpt[idx].GetFKp())/1000)
		ctx.getElementByID("txtDispenseKi").Set("value", float64(pidRpt[idx].GetFKi())/1000)
		ctx.getElementByID("txtDispenseKd").Set("value", float64(pidRpt[idx].GetFKd())/1000)
		ctx.getElementByID("txtDispenseSaturMax").Set("value", pidRpt[idx].GetSaturMax())
		ctx.getElementByID("txtDispenseSaturMin").Set("value", pidRpt[idx].GetSaturMin())
		ctx.getElementByID("txtDispensePidOffset").Set("value", pidRpt[idx].GetOffset())
		ctx.getElementByID("txtDispenseSamplingT").Set("value", pidRpt[idx].GetSamplingTimeMs())
	}

	scaleRpt := msg.GetEepromRRpt().GetScaleRpt()
	if scaleRpt != nil {
		idx, _ := strconv.ParseUint(ctx.getElementString("txtScaleIdx", "value"), 10, 32)
		if idx >= NB_OF_SCALES {
			idx = NB_OF_SCALES - 1
		}
		ctx.getElementByID("txtScaleReadingSamples").Set("value", scaleRpt[idx].GetReadingSamples())
		ctx.getElementByID("txtScaleCalibrWeight").Set("value", scaleRpt[idx].GetCalibWeightG())
		ctx.getElementByID("txtScaleCalibrWSampl").Set("value", scaleRpt[idx].GetFullCalibSamples())
		ctx.getElementByID("txtScaleTareSampl").Set("value", scaleRpt[idx].GetTareSamples())
		ctx.getElementByID("txtScaleCalibrZeroSampl").Set("value", scaleRpt[idx].GetZeroCalibSamples())
		ctx.getElementByID("txtScaleTrayWeight").Set("value", scaleRpt[idx].GetTrayWeightG())
	}

	dcMotorRpt := msg.GetEepromRRpt().GetDcmotRpt()
	if dcMotorRpt != nil {
		idx, _ := strconv.ParseUint(ctx.getElementString("txtDcMotorIdx", "value"), 10, 32)
		if idx >= NB_OF_DC_MOTORS {
			idx = NB_OF_DC_MOTORS - 1
		}
		ctx.getElementByID("txtDcMotorSpeedPerc").Set("value", dcMotorRpt[idx].GetSpeedPct())
		ctx.getElementByID("txtDcMotorDir").Set("value", dcMotorRpt[idx].GetDirection())
		ctx.getElementByID("txtDcMotorRetreatSpeed").Set("value", dcMotorRpt[idx].GetRetreatSpeedPct())
		ctx.getElementByID("txtDcMotorRetreatTime").Set("value", dcMotorRpt[idx].GetRetreatTimeMs())
	}

	massRpt := msg.GetEepromRRpt().GetMassRpt()
	if massRpt != nil {
		idx, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMassIdx", "value"), 10, 32)
		if idx >= NB_OF_MASS_SETTINGS {
			idx = NB_OF_MASS_SETTINGS - 1
		}
		ctx.getElementByID("txtMassRunsMax").Set("value", massRpt[idx].GetRunMax())
		ctx.getElementByID("txtMassDispenseTimeout").Set("value", massRpt[idx].GetDispensingTimeoutMs())
	}

	TemperatureControlRpt := msg.GetEepromRRpt().GetTemperatureRpt()
	if TemperatureControlRpt != nil {
		idx, _ := strconv.ParseUint(ctx.getElementString("txtTemperatureControlIdx", "value"), 10, 32)
		ctx.getElementByID("txtTemperatureControlSetPoint").Set("value", TemperatureControlRpt[idx].GetFTemperatureC())
		ctx.getElementByID("txtTemperatureControlTolerance").Set("value", TemperatureControlRpt[idx].GetFToleranceC())
		ctx.getElementByID("cmbTemperatureControlMode").Set("value", uint32(TemperatureControlRpt[idx].GetMode()))
	}

	IngredientRpt := msg.GetEepromRRpt().GetIngredientRpt()
	if IngredientRpt != nil {
		ctx.getElementByID("txtIngredientName").Set("value", IngredientRpt[0].GetIngredient())
	}

	TransportRpt := msg.GetEepromRRpt().GetTransportRpt()
	if TransportRpt != nil {
		idx, _ := strconv.ParseUint(ctx.getElementString("txtTransportIdx", "value"), 10, 32)
		print("transport idx")
		println(idx)
		Positions := TransportRpt[idx].GetPosition()
		if Positions != nil {
			ctx.getElementByID("txtPosition0Micro").Set("value", TransportRpt[idx].GetPosition()[0])
			ctx.getElementByID("txtPosition1Micro").Set("value", TransportRpt[idx].GetPosition()[1])
			ctx.getElementByID("txtPosition2Micro").Set("value", TransportRpt[idx].GetPosition()[2])
			ctx.getElementByID("txtPosition3Micro").Set("value", TransportRpt[idx].GetPosition()[3])
			ctx.getElementByID("txtPosition4Micro").Set("value", TransportRpt[idx].GetPosition()[4])
			ctx.getElementByID("txtPosition5Micro").Set("value", TransportRpt[idx].GetPosition()[5])
			ctx.getElementByID("txtPosition6Micro").Set("value", TransportRpt[idx].GetPosition()[6])
			ctx.getElementByID("txtPosition7Micro").Set("value", TransportRpt[idx].GetPosition()[7])
			ctx.getElementByID("txtPosition8Micro").Set("value", TransportRpt[idx].GetPosition()[8])
			ctx.getElementByID("txtPosition9Micro").Set("value", TransportRpt[idx].GetPosition()[9])
		}
		ctx.getElementByID("txtToleranceMicro").Set("value", TransportRpt[idx].GetTolerance())
	}

}

func (ctx *Ctx) EepromImport(this js.Value, i []js.Value) interface{} {

	result := js.Global().Call("confirm", "All previous dispenser data will be overwritten by incoming settings but will not be written to the EEPROM. Are you sure you want to continue?")
	if result.String() != "<boolean: true>" {
		return 1
	}

	exportData := ctx.getElementByID("eepromExportData").Get("value").String()
	println(exportData)

	rpt := &kentpb.CliToSrv{}
	err := proto.UnmarshalText(exportData, rpt)
	if err != nil {
		fmt.Println("Error unmarshaling", err)
		return 1
	}

	factoryRpt := rpt.GetEepromRRpt().GetFactoryRpt()
	if factoryRpt != nil {
		ctx.getElementByID("txtFactoryDispenserId").Set("value", factoryRpt.GetId())
		mac := factoryRpt.GetMac()
		ctx.getElementByID("txtFactoryMac").Set("value", fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]))
		ctx.getElementByID("cmbFactoryType").Set("value", int(factoryRpt.GetDeviceType()))
		ctx.getElementByID("txtFactoryHwRev").Set("value", factoryRpt.GetHwRev())
	}

	// scale
	scaleRpt := rpt.GetEepromRRpt().GetScaleRpt()
	if scaleRpt != nil {
		for idx := 0; idx < NB_OF_SCALES; idx++ {
			ctx.getElementByID("txtScaleIdx").Set("value", idx)
			ctx.getElementByID("txtScaleReadingSamples").Set("value", scaleRpt[idx].GetReadingSamples())
			ctx.getElementByID("txtScaleCalibrWeight").Set("value", scaleRpt[idx].GetCalibWeightG())
			ctx.getElementByID("txtScaleCalibrWSampl").Set("value", scaleRpt[idx].GetFullCalibSamples())
			ctx.getElementByID("txtScaleTareSampl").Set("value", scaleRpt[idx].GetTareSamples())
			ctx.getElementByID("txtScaleCalibrZeroSampl").Set("value", scaleRpt[idx].GetZeroCalibSamples())
			ctx.getElementByID("txtScaleTrayWeight").Set("value", scaleRpt[idx].GetTrayWeightG())
			ctx.ScaleSetParams(this, i)
		}
	}

	time.Sleep(600 * time.Millisecond)

	// stepper
	stepperRpt := rpt.GetEepromRRpt().GetStepperRpt()
	if stepperRpt != nil {
		for idx := 0; idx < NB_OF_STEPPERS; idx++ {
			ctx.getElementByID("txtStepperIdx").Set("value", idx)
			ctx.getElementByID("txtStepperDir").Set("value", stepperRpt[idx].GetDirection())
			ctx.getElementByID("txtStepperSpeedRps").Set("value", float64(stepperRpt[idx].GetFSpeedMaxRps())/1000)
			ctx.getElementByID("txtStepperAccelRps").Set("value", float64(stepperRpt[idx].GetFAccelRpss())/1000)
			ctx.getElementByID("txtStepperDecelRps").Set("value", float64(stepperRpt[idx].GetFDecelRpss())/1000)
			ctx.getElementByID("txtStepperHomeSpeedRps").Set("value", float64(stepperRpt[idx].GetFHomeSpeedRps())/1000)
			ctx.getElementByID("txtStepperHomeAccelRps").Set("value", float64(stepperRpt[idx].GetFHomeAccelRpss())/1000)
			ctx.getElementByID("txtStepperMaxCurrent").Set("value", stepperRpt[idx].GetCurrentMax())
			ctx.getElementByID("txtStepperMinCurrent").Set("value", stepperRpt[idx].GetCurrentMin())
			ctx.getElementByID("txtStepperHoldCurrent").Set("value", stepperRpt[idx].GetHoldCurrent())
			ctx.getElementByID("txtStepperRetreatSpeed").Set("value", stepperRpt[idx].GetRetreatSpeedPct())
			ctx.getElementByID("txtStepperRetreatAngle").Set("value", stepperRpt[idx].GetRetreatAngle())
			ctx.StepperSetParams(this, i)
			time.Sleep(600 * time.Millisecond)
		}
	}

	// dc motor
	dcMotorRpt := rpt.GetEepromRRpt().GetDcmotRpt()
	if dcMotorRpt != nil {
		for idx := 0; idx < NB_OF_DC_MOTORS; idx++ {
			ctx.getElementByID("txtDcMotorIdx").Set("value", idx)
			ctx.getElementByID("txtDcMotorSpeedPerc").Set("value", dcMotorRpt[idx].GetSpeedPct())
			ctx.getElementByID("txtDcMotorDir").Set("value", dcMotorRpt[idx].GetDirection())
			ctx.getElementByID("txtDcMotorRetreatSpeed").Set("value", dcMotorRpt[idx].GetRetreatSpeedPct())
			ctx.getElementByID("txtDcMotorRetreatTime").Set("value", dcMotorRpt[idx].GetRetreatTimeMs())
			ctx.DcMotorSetParams(this, i)
		}
	}

	time.Sleep(600 * time.Millisecond)

	// pid
	pidRpt := rpt.GetEepromRRpt().GetPidRpt()
	if pidRpt != nil {
		for idx := 0; idx < NB_OF_PID_SETTINGS; idx++ {
			ctx.getElementByID("txtPidIdx").Set("value", idx)
			ctx.getElementByID("txtDispenseKp").Set("value", float64(pidRpt[idx].GetFKp())/1000)
			ctx.getElementByID("txtDispenseKi").Set("value", float64(pidRpt[idx].GetFKi())/1000)
			ctx.getElementByID("txtDispenseKd").Set("value", float64(pidRpt[idx].GetFKd())/1000)
			ctx.getElementByID("txtDispenseSaturMax").Set("value", pidRpt[idx].GetSaturMax())
			ctx.getElementByID("txtDispenseSaturMin").Set("value", pidRpt[idx].GetSaturMin())
			ctx.getElementByID("txtDispensePidOffset").Set("value", pidRpt[idx].GetOffset())
			ctx.getElementByID("txtDispenseSamplingT").Set("value", pidRpt[idx].GetSamplingTimeMs())
			ctx.PidSetParams(this, i)
		}
	}

	time.Sleep(600 * time.Millisecond)

	// mass
	massRpt := rpt.GetEepromRRpt().GetMassRpt()
	if massRpt != nil {
		for idx := 0; idx < NB_OF_MASS_SETTINGS; idx++ {
			ctx.getElementByID("txtDispenseMassIdx").Set("value", idx)
			ctx.getElementByID("txtMassRunsMax").Set("value", massRpt[idx].GetRunMax())
			ctx.getElementByID("txtMassDispenseTimeout").Set("value", massRpt[idx].GetDispensingTimeoutMs())
			ctx.MassSetParams(this, i)
		}
	}

	time.Sleep(600 * time.Millisecond)

	// temp control
	temperatureControlRpt := rpt.GetEepromRRpt().GetTemperatureRpt()
	if temperatureControlRpt != nil {
		for idx := 0; idx < NB_OF_TEMP_CONTROLLERS; idx++ {
			ctx.getElementByID("txtTemperatureControlIdx").Set("value", idx)
			ctx.getElementByID("txtTemperatureControlSetPoint").Set("value", temperatureControlRpt[idx].GetFTemperatureC())
			ctx.getElementByID("txtTemperatureControlTolerance").Set("value", temperatureControlRpt[idx].GetFToleranceC())
			ctx.getElementByID("cmbTemperatureControlMode").Set("value", uint32(temperatureControlRpt[idx].GetMode()))
			ctx.TemperatureControlSetParams(this, i)
		}
	}

	time.Sleep(600 * time.Millisecond)

	// ingredient
	ingredientRpt := rpt.GetEepromRRpt().GetIngredientRpt()
	if ingredientRpt != nil {
		ctx.getElementByID("txtIngredientName").Set("value", ingredientRpt[0].GetIngredient())
		ctx.IngredientSetParams(this, i)
	}

	time.Sleep(600 * time.Millisecond)

	// transport positions
	transportRpt := rpt.GetEepromRRpt().GetTransportRpt()
	if transportRpt != nil {
		for idx := 0; idx < NB_OF_TRANSPORTS; idx++ {
			ctx.getElementByID("txtTransportIdx").Set("value", idx)
			Positions := transportRpt[idx].GetPosition()

			if Positions != nil {
				ctx.getElementByID("txtPosition0Micro").Set("value", transportRpt[idx].GetPosition()[0])
				ctx.getElementByID("txtPosition1Micro").Set("value", Positions[1])
				ctx.getElementByID("txtPosition2Micro").Set("value", Positions[2])
				ctx.getElementByID("txtPosition3Micro").Set("value", Positions[3])
				ctx.getElementByID("txtPosition4Micro").Set("value", Positions[4])
				ctx.getElementByID("txtPosition5Micro").Set("value", Positions[5])
				ctx.getElementByID("txtPosition6Micro").Set("value", Positions[6])
				ctx.getElementByID("txtPosition7Micro").Set("value", Positions[7])
				ctx.getElementByID("txtPosition8Micro").Set("value", Positions[8])
				ctx.getElementByID("txtPosition9Micro").Set("value", Positions[9])
			}
			if Positions == nil {
				print("transport idx")
				println(idx)
				print("nil")
			}
			ctx.getElementByID("txtToleranceMicro").Set("value", transportRpt[idx].GetTolerance())

			ctx.TransportPosSetParams(this, i)
			time.Sleep(600 * time.Millisecond)
		}
	}

	return 1
}

/*
ClearLog -
*/
func (ctx *Ctx) ClearLog(this js.Value, i []js.Value) interface{} {
	ctx.getElementByID("txtLogArea").Set("value", "")
	return 1
}

/*
ClearPidLog -
*/
func (ctx *Ctx) ClearPidLog(this js.Value, i []js.Value) interface{} {

	pidAreaDefaultValue := "Run" + "\t" + "Loop" + "\t" + "t" + "\t" + "Sp" + "\t" + "Cv" + "\t" + "Err" + "\t" + "Int" + "\t" + "Der" + "\t" + "P" + "\t" + "I" + "\t" + "D" + "\t" + "Pv\n"

	ctx.getElementByID("txtPidAreaTitle").Set("value", pidAreaDefaultValue)
	ctx.getElementByID("txtPidArea").Set("value", "")
	return 1
}

/*
Connect -
*/
func (ctx *Ctx) Connect(this js.Value, i []js.Value) interface{} {

	if ctx.wsConn {
		ctx.appendToLog("Already connected!")
		return 1
	}

	ctx.appendToLog("Connecting..")

	ip := ctx.getElementString("txtBrokerIp", "value")
	port := ctx.getElementString("txtBrokerPort", "value")
	wsString := "ws://" + string(ip) + ":" + string(port) + "/ws"

	ctx.wsSrv = js.Global().Get("WebSocket").New(wsString)
	ctx.wsConn = true

	ctx.wsSrv.Call("addEventListener", "open", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ctx.appendToLog("Connected!")
		return nil
	}))

	ctx.wsSrv.Call("addEventListener", "close", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ctx.wsConn = false
		return nil
	}))

	ctx.wsSrv.Call("addEventListener", "message", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ctx.receiveFromWs(args[0].Get("data"))
		return nil
	}))

	ctx.wsSrv.Call("addEventListener", "error", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ctx.wsConn = false
		ctx.appendToLog("Connection failed!")
		return nil
	}))
	return 1
}

/*
Disconnect -
*/
func (ctx *Ctx) Disconnect(this js.Value, i []js.Value) interface{} {

	ctx.wsSrv.Call("close")
	ctx.wsConn = false
	ctx.appendToLog("Disconnected!")

	return 1
}

/*
DispenserReboot -
*/
func (ctx *Ctx) DispenserReboot(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	result := js.Global().Call("confirm", "All the unsaved changes will be lost. Are you sure you want to continue?")
	if result.String() == "<boolean: true>" {
		req := &kentpb.SrvToCli{
			ReqOneof: &kentpb.SrvToCli_RebootReq{},
		}
		ctx.sendToWs(ctx.getDispenserID(), req)
	}
	return 1
}

/*
EepromRead -
*/
func (ctx *Ctx) EepromRead(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromRReq{},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
EepromWrite -
*/
func (ctx *Ctx) EepromWrite(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	result := js.Global().Call("confirm", "All the previous dispenser data will be overwritten by the current settings. Are you sure you want to continue?")
	if result.String() == "<boolean: true>" {
		ctx.ScaleSetParams(this, i)
		ctx.MassSetParams(this, i)
		ctx.PidSetParams(this, i)
		ctx.StepperSetParams(this, i)
		ctx.DcMotorSetParams(this, i)
		ctx.TemperatureControlSetParams(this, i)
		ctx.IngredientSetParams(this, i)
		ctx.TransportPosSetParams(this, i)

		req := &kentpb.SrvToCli{
			ReqOneof: &kentpb.SrvToCli_EepromWReq{},
		}
		ctx.sendToWs(ctx.getDispenserID(), req)
	}

	return 1
}

/*
UpgradeFirmware -
*/
func (ctx *Ctx) UpgradeFirmware(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	firmwareUrl := ctx.getElementString("txtFirmwareUrl", "value")
	firmwareType, _ := strconv.ParseUint(ctx.getElementString("cmbFirmwareType", "value"), 10, 32)

	ctx.appendToLog(firmwareUrl)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_UpgradeFwReq{
			&kentpb.UpgradeFirmwareRequest{
				FwType: kentpb.UpgradeFirmwareRequest_FirmwareType(firmwareType),
				Url:    firmwareUrl,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
FactoryChange -
*/
func (ctx *Ctx) FactoryChange(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	factoryDispenserID := ctx.getElementString("txtFactoryDispenserId", "value")
	factoryMac := ctx.getElementString("txtFactoryMac", "value")
	factoryDispenserType, _ := strconv.ParseUint(ctx.getElementString("cmbFactoryType", "value"), 10, 32)
	factoryHwRev, _ := strconv.ParseUint(ctx.getElementString("txtFactoryHwRev", "value"), 10, 32)

	var macBytes [6]byte
	fmt.Sscanf(factoryMac, "%02X:%02X:%02X:%02X:%02X:%02X", &macBytes[0], &macBytes[1], &macBytes[2], &macBytes[3], &macBytes[4], &macBytes[5])

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromFactoryReq{
			&kentpb.EepromFactoryData{
				Mac:        []byte{macBytes[0], macBytes[1], macBytes[2], macBytes[3], macBytes[4], macBytes[5]},
				Id:         factoryDispenserID,
				DeviceType: kentpb.EepromFactoryData_DeviceType(factoryDispenserType),
				HwRev:      uint32(factoryHwRev),
			},
		},
	}

	result := js.Global().Call("confirm", "Vital dispenser data will be changed. You will need to write the parameters to EEPROM and reboot the dispenser to apply the changes. Are you sure you want to continue?")
	if result.String() == "<boolean: true>" {
		ctx.sendToWs(ctx.getDispenserID(), req)
	}

	return 1
}

/*
ScaleRead -
*/
func (ctx *Ctx) ScaleRead(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	scaleIdx := ctx.getElementString("txtScaleIdx", "value")
	idx, _ := strconv.ParseUint(scaleIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DbgScaleReadReq{
			&kentpb.DbgScaleRequest{
				Idx: uint32(idx),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1

}

/*
ScaleTare -
*/
func (ctx *Ctx) ScaleTare(this js.Value, i []js.Value) interface{} {

	ctx.ScaleSetParams(this, i)

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	scaleIdx := ctx.getElementString("txtScaleIdx", "value")
	idx, _ := strconv.ParseUint(scaleIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DbgScaleTareReq{
			&kentpb.DbgScaleRequest{
				Idx: uint32(idx),
			},
		},
	}

	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
ScaleCalibFull -
*/
func (ctx *Ctx) ScaleCalibFull(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	scaleIdx := ctx.getElementString("txtScaleIdx", "value")
	idx, _ := strconv.ParseUint(scaleIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_ScaleCalibReq{
			&kentpb.ScaleCalibrateRequest{
				Idx:       uint32(idx),
				CalibType: kentpb.ScaleCalibrateRequest_CT_FULL_CALIBRATION,
			},
		},
	}

	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
ScaleCalibFull -
*/
func (ctx *Ctx) ScaleCalibZero(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	scaleIdx := ctx.getElementString("txtScaleIdx", "value")
	idx, _ := strconv.ParseUint(scaleIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_ScaleCalibReq{
			&kentpb.ScaleCalibrateRequest{
				Idx:       uint32(idx),
				CalibType: kentpb.ScaleCalibrateRequest_CT_ZERO_CALIBRATION,
			},
		},
	}

	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
ScaleSetParams -
*/
func (ctx *Ctx) ScaleSetParams(this js.Value, i []js.Value) interface{} {

	scaleIdx := ctx.getElementString("txtScaleIdx", "value")
	readingSamples := ctx.getElementString("txtScaleReadingSamples", "value")
	calibWeight := ctx.getElementString("txtScaleCalibrWeight", "value")
	calibWSampl := ctx.getElementString("txtScaleCalibrWSampl", "value")
	tareSampl := ctx.getElementString("txtScaleTareSampl", "value")
	zeroSampl := ctx.getElementString("txtScaleCalibrZeroSampl", "value")
	trayWeight := ctx.getElementString("txtScaleTrayWeight", "value")

	idx, _ := strconv.ParseUint(scaleIdx, 10, 32)
	cs, _ := strconv.ParseUint(calibWSampl, 10, 32)
	ts, _ := strconv.ParseUint(tareSampl, 10, 32)
	rs, _ := strconv.ParseUint(readingSamples, 10, 32)
	cw, _ := strconv.ParseUint(calibWeight, 10, 32)
	zs, _ := strconv.ParseUint(zeroSampl, 10, 32)
	tw, _ := strconv.ParseUint(trayWeight, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromScaleReq{
			&kentpb.EepromScaleData{
				Idx:              uint32(idx),
				FullCalibSamples: uint32(cs),
				TareSamples:      uint32(ts),
				ReadingSamples:   uint32(rs),
				CalibWeightG:     uint32(cw),
				ZeroCalibSamples: uint32(zs),
				TrayWeightG:      uint32(tw),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
HopperRead -
*/
func (ctx *Ctx) HopperRead(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	hopperIdx := ctx.getElementString("txtHopperIdx", "value")
	idx, _ := strconv.ParseUint(hopperIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserDbgHopperReadReq{
			&kentpb.DbgScaleRequest{
				Idx: uint32(idx),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1

}

/*
HopperCalibrationOffset -
*/
func (ctx *Ctx) HopperCalibrationOffset(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	hopperIdx := ctx.getElementString("txtHopperIdx", "value")
	idx, _ := strconv.ParseUint(hopperIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserDbgHopperCalibOffsetReq{
			&kentpb.DbgScaleRequest{
				Idx: uint32(idx),
			},
		},
	}

	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
StepperRotate -
*/
func (ctx *Ctx) StepperRotate(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}
	stepperIdx := ctx.getElementString("txtStepperIdx", "value")
	stepperRotateAngle, _ := strconv.ParseInt(ctx.getElementString("txtStepperAngle", "value"), 10, 32)
	stepperCurrent, _ := strconv.ParseUint(ctx.getElementString("txtStepperCurrent", "value"), 10, 32)

	idx, _ := strconv.ParseFloat(stepperIdx, 64)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DbgStepperRotateReq{
			&kentpb.DbgStepperRequest{
				Idx:     uint32(idx),
				Current: uint32(stepperCurrent),
				Angle:   int32(stepperRotateAngle),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
StepperSpin -
*/
func (ctx *Ctx) StepperSpin(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	stepperIdx := ctx.getElementString("txtStepperIdx", "value")
	stepperCurrent, _ := strconv.ParseUint(ctx.getElementString("txtStepperCurrent", "value"), 10, 32)

	idx, _ := strconv.ParseFloat(stepperIdx, 64)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DbgStepperSpinReq{
			&kentpb.DbgStepperRequest{
				Idx:     uint32(idx),
				Current: uint32(stepperCurrent),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
StepperStop -
*/
func (ctx *Ctx) StepperStop(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	stepperIdx := ctx.getElementString("txtStepperIdx", "value")
	idx, _ := strconv.ParseFloat(stepperIdx, 64)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DbgStepperStopReq{
			&kentpb.DbgStepperRequest{
				Idx: uint32(idx),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
StepperSetParams -
*/
func (ctx *Ctx) StepperSetParams(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	stepperIdx := ctx.getElementString("txtStepperIdx", "value")
	stepperMaxSpeedRps := ctx.getElementString("txtStepperSpeedRps", "value")
	stepperAccelRps := ctx.getElementString("txtStepperAccelRps", "value")
	stepperDecelRps := ctx.getElementString("txtStepperDecelRps", "value")
	stepperHomeSpeedRps := ctx.getElementString("txtStepperHomeSpeedRps", "value")
	stepperHomeAccelRps := ctx.getElementString("txtStepperHomeAccelRps", "value")
	stepperDirection := ctx.getElementString("txtStepperDir", "value")
	stepperMaxCurrent := ctx.getElementString("txtStepperMaxCurrent", "value")
	stepperMinCurrent := ctx.getElementString("txtStepperMinCurrent", "value")
	stepperHoldCurrent := ctx.getElementString("txtStepperHoldCurrent", "value")
	massRetreatSpeed := ctx.getElementString("txtStepperRetreatSpeed", "value")
	massRetreatAngle := ctx.getElementString("txtStepperRetreatAngle", "value")

	idx, _ := strconv.ParseFloat(stepperIdx, 64)
	maxSpeed, _ := strconv.ParseFloat(stepperMaxSpeedRps, 64)
	maxSpeed *= 1000
	accl, _ := strconv.ParseFloat(stepperAccelRps, 64)
	accl *= 1000
	decl, _ := strconv.ParseFloat(stepperDecelRps, 64)
	decl *= 1000
	homeSpeed, _ := strconv.ParseFloat(stepperHomeSpeedRps, 64)
	homeSpeed *= 1000
	homeAccl, _ := strconv.ParseFloat(stepperHomeAccelRps, 64)
	homeAccl *= 1000
	dir, _ := strconv.ParseUint(stepperDirection, 10, 32)
	maxCur, _ := strconv.ParseUint(stepperMaxCurrent, 10, 32)
	minCur, _ := strconv.ParseUint(stepperMinCurrent, 10, 32)
	holdCur, _ := strconv.ParseUint(stepperHoldCurrent, 10, 32)
	retreatSpeed, _ := strconv.ParseUint(massRetreatSpeed, 10, 32)
	retreatAngle, _ := strconv.ParseInt(massRetreatAngle, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromStepperReq{
			&kentpb.EepromStepperData{
				Idx:             uint32(idx),
				Direction:       uint32(dir),
				FSpeedMaxRps:    uint32(maxSpeed),
				FAccelRpss:      uint32(accl),
				CurrentMax:      uint32(maxCur),
				CurrentMin:      uint32(minCur),
				RetreatSpeedPct: uint32(retreatSpeed),
				RetreatAngle:    int32(retreatAngle),
				HoldCurrent:     uint32(holdCur),
				FDecelRpss:      uint32(decl),
				FHomeSpeedRps:   uint32(homeSpeed),
				FHomeAccelRpss:  uint32(homeAccl),
			},
		},
	}

	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
VibratorRun -
*/
func (ctx *Ctx) VibratorRun(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	vibratorAmplPerc, _ := strconv.ParseUint(ctx.getElementString("txtVibratorAmplitube", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserDbgVibratorRunReq{
			&kentpb.DispenserDbgVibratorRequest{
				Idx:          0,
				AmplitubePct: uint32(vibratorAmplPerc),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
VibratorStop -
*/
func (ctx *Ctx) VibratorStop(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserDbgVibratorStopReq{
			&kentpb.DispenserDbgVibratorRequest{
				Idx: 0,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
DcMotorSpin -
*/
func (ctx *Ctx) DcMotorSpin(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtDcMotorIdx", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DbgDcmotorSpinReq{
			&kentpb.DbgDcMotorRequest{
				Idx: uint32(idx),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
DcMotorStop -
*/
func (ctx *Ctx) DcMotorStop(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtDcMotorIdx", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DbgDcmotorStopReq{
			&kentpb.DbgDcMotorRequest{
				Idx: uint32(idx),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
DcMotorSetParams -
*/
func (ctx *Ctx) DcMotorSetParams(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	dcMotorDirection := ctx.getElementString("txtDcMotorDir", "value")
	dcMotorSpeedPerc := ctx.getElementString("txtDcMotorSpeedPerc", "value")
	dcMotorRetreatSpeed := ctx.getElementString("txtDcMotorRetreatSpeed", "value")
	dcMotorRetreatTime := ctx.getElementString("txtDcMotorRetreatTime", "value")
	dcMotorIdx := ctx.getElementString("txtDcMotorIdx", "value")

	dir, _ := strconv.ParseUint(dcMotorDirection, 10, 32)
	speedPerc, _ := strconv.ParseUint(dcMotorSpeedPerc, 10, 32)
	retreatSpeed, _ := strconv.ParseUint(dcMotorRetreatSpeed, 10, 32)
	retreatTime, _ := strconv.ParseUint(dcMotorRetreatTime, 10, 32)
	idx, _ := strconv.ParseUint(dcMotorIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromDcmotorReq{
			&kentpb.EepromDcMotorData{
				Idx:             uint32(idx),
				Direction:       uint32(dir),
				SpeedPct:        uint32(speedPerc),
				RetreatSpeedPct: uint32(retreatSpeed),
				RetreatTimeMs:   uint32(retreatTime),
			},
		},
	}

	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
DispenseMass -
*/
func (ctx *Ctx) DispenseMass(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMassIdx", "value"), 10, 32)
	massG, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMass", "value"), 10, 32)
	correctionG, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMassCorrection", "value"), 10, 32)
	simulTimeMs, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMassSimT", "value"), 10, 32)
	pidDbg, _ := strconv.ParseUint(ctx.getElementString("txtDispensePidDbg", "value"), 10, 32)
	cookTimeMs, _ := strconv.ParseUint(ctx.getElementString("txtProcessCookTime", "value"), 10, 32)

	if cookTimeMs == 0 {
		ctx.appendToLog("Cook time cannot be 0!")
		return 1
	}

	ctx.ClearPidLog(this, i)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserProcessReq{
			&kentpb.DispenserProcessRequest{
				Idx:              uint32(idx),
				MassMg:           uint32(massG) * 1000,
				MassCorrectionMg: int32(correctionG) * 1000,
				SimulationTimeMs: uint32(simulTimeMs),
				PidDbg:           uint32(pidDbg),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1

}

/*
MassSetParams -
*/
func (ctx *Ctx) MassSetParams(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	runsMax := ctx.getElementString("txtMassRunsMax", "value")
	massIdx := ctx.getElementString("txtDispenseMassIdx", "value")
	dispTimeout := ctx.getElementString("txtMassDispenseTimeout", "value")

	max, _ := strconv.ParseUint(runsMax, 10, 32)
	timeout, _ := strconv.ParseInt(dispTimeout, 10, 32)
	idx, _ := strconv.ParseUint(massIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserEepromMassReq{
			&kentpb.DispenserEepromMassData{
				Idx:                 uint32(idx),
				RunMax:              uint32(max),
				DispensingTimeoutMs: int32(timeout),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
CookToRate -
*/
func (ctx *Ctx) CookToRate(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMassIdx", "value"), 10, 32)
	massG, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMass", "value"), 10, 32)
	correctionG, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMassCorrection", "value"), 10, 32)
	simulTimeMs, _ := strconv.ParseUint(ctx.getElementString("txtDispenseMassSimT", "value"), 10, 32)
	cookTimeMs, _ := strconv.ParseUint(ctx.getElementString("txtProcessCookTime", "value"), 10, 32)
	rateMs, _ := strconv.ParseUint(ctx.getElementString("txtRateTime", "value"), 10, 32)
	procMode, _ := strconv.ParseUint(ctx.getElementString("cmbProcMode", "value"), 10, 32)
	dripTimeMs, _ := strconv.ParseUint(ctx.getElementString("txtDripTime", "value"), 10, 32)
	shakeTimeMs, _ := strconv.ParseUint(ctx.getElementString("txtShakeTime", "value"), 10, 32)
	nbShakes, _ := strconv.ParseUint(ctx.getElementString("txtNbShakes", "value"), 10, 32)
	preemptive, _ := strconv.ParseUint(ctx.getElementString("cmbPreemptive", "value"), 10, 32)

	ctx.ClearPidLog(this, i)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerProcessReq{
			&kentpb.FryerProcessRequest{
				FryPositionIdx:   uint32(idx),
				MassMg:           uint32(massG) * 1000,
				MassCorrectionMg: int32(correctionG) * 1000,
				SimulationTimeMs: uint32(simulTimeMs),
				CookingTimeMs:    uint32(cookTimeMs),
				OrderRateMs:      uint32(rateMs),
				ProcessingMode:   kentpb.FryerProcessingMode(procMode),
				DripTimeMs:       uint32(dripTimeMs),
				NumberOfShakes:   uint32(nbShakes),
				ShakeTimeMs:      uint32(shakeTimeMs),
				PreemptiveFlag:	  uint32(preemptive) == 1,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
FreezerActivate -
*/
func (ctx *Ctx) FreezerActivate(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtFreezerIdx", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerFreezerReq{
			&kentpb.FryerFreezerRequest{
				Idx:     uint32(idx),
				Enabled: true,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
FreezerDeactivate -
*/
func (ctx *Ctx) FreezerDeactivate(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtFreezerIdx", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerFreezerReq{
			&kentpb.FryerFreezerRequest{
				Idx:     uint32(idx),
				Enabled: false,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
FreezerUnlock -
*/
func (ctx *Ctx) FreezerUnlock(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtFreezerIdx", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerFreezerDrawerUnlockReq{
			&kentpb.FryerFreezerDrawerUnlockRequest{
				Idx:    uint32(idx),
				Unlock: true,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
HotHoldActivate -
*/
func (ctx *Ctx) HotHoldActivate(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtHotHoldIdx", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerHotHoldReq{
			&kentpb.FryerHotHoldRequest{
				Idx:     uint32(idx),
				Enabled: true,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
HotHoldDeactivate -
*/
func (ctx *Ctx) HotHoldDeactivate(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	idx, _ := strconv.ParseUint(ctx.getElementString("txtHotHoldIdx", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerHotHoldReq{
			&kentpb.FryerHotHoldRequest{
				Idx:     uint32(idx),
				Enabled: false,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
ChangeOpMode -
*/
func (ctx *Ctx) ChangeOpMode(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	state, _ := strconv.ParseUint(ctx.getElementString("cmbOpMode", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerSetOperatingStateReq{
			&kentpb.FryerSetOperatingStateRequest{
				State: kentpb.FryerOperatingState(state),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
TransportPosSetParams
*/
func (ctx *Ctx) TransportPosSetParams(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	transportIdx, _ := strconv.ParseUint(ctx.getElementString("txtTransportIdx", "value"), 10, 32)
	position0, _ := strconv.ParseInt(ctx.getElementString("txtPosition0Micro", "value"), 10, 32)
	position1, _ := strconv.ParseInt(ctx.getElementString("txtPosition1Micro", "value"), 10, 32)
	position2, _ := strconv.ParseInt(ctx.getElementString("txtPosition2Micro", "value"), 10, 32)
	position3, _ := strconv.ParseInt(ctx.getElementString("txtPosition3Micro", "value"), 10, 32)
	position4, _ := strconv.ParseInt(ctx.getElementString("txtPosition4Micro", "value"), 10, 32)
	position5, _ := strconv.ParseInt(ctx.getElementString("txtPosition5Micro", "value"), 10, 32)
	position6, _ := strconv.ParseInt(ctx.getElementString("txtPosition6Micro", "value"), 10, 32)
	position7, _ := strconv.ParseInt(ctx.getElementString("txtPosition7Micro", "value"), 10, 32)
	position8, _ := strconv.ParseInt(ctx.getElementString("txtPosition8Micro", "value"), 10, 32)
	position9, _ := strconv.ParseInt(ctx.getElementString("txtPosition9Micro", "value"), 10, 32)
	tolerance, _ := strconv.ParseInt(ctx.getElementString("txtToleranceMicro", "value"), 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_FryerEepromPositionsReq{
			&kentpb.EepromPositionsRequest{
				Idx: uint32(transportIdx),
				Position: []int32{
					int32(position0), int32(position1),
					int32(position2), int32(position3),
					int32(position4), int32(position5),
					int32(position6), int32(position7),
					int32(position8), int32(position9)},
				Tolerance: int32(tolerance),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
PidSetParams -
*/
func (ctx *Ctx) PidSetParams(this js.Value, i []js.Value) interface{} {

	pidKp := ctx.getElementString("txtDispenseKp", "value")
	pidKi := ctx.getElementString("txtDispenseKi", "value")
	pidKd := ctx.getElementString("txtDispenseKd", "value")
	pidSaturMax := ctx.getElementString("txtDispenseSaturMax", "value")
	pidSaturMin := ctx.getElementString("txtDispenseSaturMin", "value")
	pidOffset := ctx.getElementString("txtDispensePidOffset", "value")
	samplingTime := ctx.getElementString("txtDispenseSamplingT", "value")
	pidIdx := ctx.getElementString("txtPidIdx", "value")

	kp, _ := strconv.ParseFloat(pidKp, 64)
	kp *= 1000
	ki, _ := strconv.ParseFloat(pidKi, 64)
	ki *= 1000
	kd, _ := strconv.ParseFloat(pidKd, 64)
	kd *= 1000
	max, _ := strconv.ParseUint(pidSaturMax, 10, 32)
	min, _ := strconv.ParseUint(pidSaturMin, 10, 32)
	offset, _ := strconv.ParseUint(pidOffset, 10, 32)
	sampTime, _ := strconv.ParseUint(samplingTime, 10, 32)
	idx, _ := strconv.ParseUint(pidIdx, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromPidReq{
			&kentpb.EepromPidData{
				Idx:            uint32(idx),
				FKp:            int32(kp),
				FKi:            int32(ki),
				FKd:            int32(kd),
				SaturMax:       int32(max),
				SaturMin:       int32(min),
				DeltaT:         1,
				Offset:         int32(offset),
				SamplingTimeMs: uint32(sampTime),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
GetState -
*/
func (ctx *Ctx) GetState(this js.Value, i []js.Value) interface{} {
	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_StateReq{
			&kentpb.StateRequest{
				DummyField: 0,
			},
		},
	}

	ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
DrawerLock -
*/
func (ctx *Ctx) DrawerLock(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	// index := ctx.getElementString("txtPassIdx", "value")
	// idx, _ := strconv.ParseUint(index, 10, 32)

	// req := &kentpb.SrvToCli{
	// 	ReqOneof: &kentpb.SrvToCli_PassLockReq{
	// 		&kentpb.PassLockRequest{
	// 			Idx:  uint32(idx),
	// 			Lock: true,
	// 		},
	// 	},
	// }

	//ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

/*
DrawerUnlock -
*/
func (ctx *Ctx) DrawerUnlock(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	// index := ctx.getElementString("txtPassIdx", "value")
	// idx, _ := strconv.ParseUint(index, 10, 32)

	// req := &kentpb.SrvToCli{
	// 	ReqOneof: &kentpb.SrvToCli_PassLockReq{
	// 		&kentpb.PassLockRequest{
	// 			Idx:  uint32(idx),
	// 			Lock: false,
	// 		},
	// 	},
	// }

	//ctx.sendToWs(ctx.getDispenserID(), req)
	return 1
}

func (ctx *Ctx) TemperatureControlEnable(this js.Value, i []js.Value) interface{} {

	index := ctx.getElementString("txtTemperatureControlIdx", "value")
	idx, _ := strconv.ParseUint(index, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserTemperatureCtrlReq{
			&kentpb.DispenserTemperatureControlRequest{
				Idx:     uint32(idx),
				Enabled: true,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1

}

func (ctx *Ctx) TemperatureControlDisable(this js.Value, i []js.Value) interface{} {
	index := ctx.getElementString("txtTemperatureControlIdx", "value")
	idx, _ := strconv.ParseUint(index, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserTemperatureCtrlReq{
			&kentpb.DispenserTemperatureControlRequest{
				Idx:     uint32(idx),
				Enabled: false,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1

}

func (ctx *Ctx) TemperatureControlSetParams(this js.Value, i []js.Value) interface{} {
	index := ctx.getElementString("txtTemperatureControlIdx", "value")
	setPoint := ctx.getElementString("txtTemperatureControlSetPoint", "value")
	tolerance := ctx.getElementString("txtTemperatureControlTolerance", "value")
	mode := ctx.getElementString("cmbTemperatureControlMode", "value")

	idx, _ := strconv.ParseUint(index, 10, 32)
	sp, _ := strconv.ParseUint(setPoint, 10, 32)
	t, _ := strconv.ParseUint(tolerance, 10, 32)
	m, _ := strconv.ParseUint(mode, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromTemperatureReq{
			&kentpb.EepromTemperatureControlData{
				Idx:           uint32(idx),
				FTemperatureC: int32(sp),
				FToleranceC:   uint32(t),
				Mode:          kentpb.EepromTemperatureControlData_TemperatureControlMode(m),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

func (ctx *Ctx) AgitatorRun(this js.Value, i []js.Value) interface{} {
	index := ctx.getElementString("txtAgitatorIdx", "value")
	idx, _ := strconv.ParseUint(index, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserAgitationReq{
			&kentpb.DispenserAgitationRequest{
				Idx:     uint32(idx),
				Enabled: true,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

func (ctx *Ctx) AgitatorStop(this js.Value, i []js.Value) interface{} {
	index := ctx.getElementString("txtAgitatorIdx", "value")
	idx, _ := strconv.ParseUint(index, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_DispenserAgitationReq{
			&kentpb.DispenserAgitationRequest{
				Idx:     uint32(idx),
				Enabled: false,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

func (ctx *Ctx) IngredientSetParams(this js.Value, i []js.Value) interface{} {
	ingredient := ctx.getElementString("txtIngredientName", "value")

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_EepromIngredientReq{
			&kentpb.EepromIngredientData{
				Idx:        0,
				Ingredient: ingredient,
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1
}

/*
TransportMove -
*/
func (ctx *Ctx) TransportMove(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	transportIdx := ctx.getElementString("txtTransportIdx", "value")
	transportToPos := ctx.getElementString("txtTransportMoveToPos", "value")
	transportFromPos := ctx.getElementString("txtTransportMoveFromPos", "value")
	idx, _ := strconv.ParseUint(transportIdx, 10, 32)
	toPos, _ := strconv.ParseUint(transportToPos, 10, 32)
	fromPos, _ := strconv.ParseUint(transportFromPos, 10, 32)

	req := &kentpb.SrvToCli{
		ReqOneof: &kentpb.SrvToCli_TransportMoveReq{
			&kentpb.DbgTransportMoveRequest{
				Idx:  uint32(idx),
				To:   kentpb.DbgTransportMoveRequest_MovePosition(toPos),
				From: kentpb.DbgTransportMoveRequest_MovePosition(fromPos),
			},
		},
	}
	ctx.sendToWs(ctx.getDispenserID(), req)

	return 1

}

func (ctx *Ctx) StopAllSteppers(this js.Value, i []js.Value) interface{} {

	if !ctx.wsConn {
		ctx.appendToLog("Not Connected to Broker!")
		return 1
	}

	for i := 0; i < 6; i++ {
		req := &kentpb.SrvToCli{
			ReqOneof: &kentpb.SrvToCli_DbgStepperStopReq{
				&kentpb.DbgStepperRequest{
					Idx: uint32(i),
				},
			},
		}
		ctx.sendToWs(ctx.getDispenserID(), req)
	}
	return 1
}

func (ctx *Ctx) registerCallbacks() {
	js.Global().Set("Connect", js.FuncOf(ctx.Connect))
	js.Global().Set("Disconnect", js.FuncOf(ctx.Disconnect))

	js.Global().Set("DispenserReboot", js.FuncOf(ctx.DispenserReboot))
	js.Global().Set("FactoryChange", js.FuncOf(ctx.FactoryChange))
	js.Global().Set("EepromWrite", js.FuncOf(ctx.EepromWrite))
	js.Global().Set("EepromRead", js.FuncOf(ctx.EepromRead))
	js.Global().Set("EepromImport", js.FuncOf(ctx.EepromImport))
	js.Global().Set("UpgradeFirmware", js.FuncOf(ctx.UpgradeFirmware))

	js.Global().Set("ScaleRead", js.FuncOf(ctx.ScaleRead))
	js.Global().Set("ScaleTare", js.FuncOf(ctx.ScaleTare))
	js.Global().Set("ScaleCalibFull", js.FuncOf(ctx.ScaleCalibFull))
	js.Global().Set("ScaleCalibZero", js.FuncOf(ctx.ScaleCalibZero))
	js.Global().Set("ScaleSetParams", js.FuncOf(ctx.ScaleSetParams))

	js.Global().Set("HopperRead", js.FuncOf(ctx.HopperRead))
	js.Global().Set("HopperCalibrationOffset", js.FuncOf(ctx.HopperCalibrationOffset))

	js.Global().Set("StepperRotate", js.FuncOf(ctx.StepperRotate))
	js.Global().Set("StepperSpin", js.FuncOf(ctx.StepperSpin))
	js.Global().Set("StepperStop", js.FuncOf(ctx.StepperStop))
	js.Global().Set("StepperSetParams", js.FuncOf(ctx.StepperSetParams))

	js.Global().Set("VibratorRun", js.FuncOf(ctx.VibratorRun))
	js.Global().Set("VibratorStop", js.FuncOf(ctx.VibratorStop))

	js.Global().Set("DcMotorSpin", js.FuncOf(ctx.DcMotorSpin))
	js.Global().Set("DcMotorStop", js.FuncOf(ctx.DcMotorStop))
	js.Global().Set("DcMotorSetParams", js.FuncOf(ctx.DcMotorSetParams))

	js.Global().Set("DispenseMass", js.FuncOf(ctx.DispenseMass))
	js.Global().Set("MassSetParams", js.FuncOf(ctx.MassSetParams))
	js.Global().Set("CookToRate", js.FuncOf(ctx.CookToRate))
	js.Global().Set("FreezerActivate", js.FuncOf(ctx.FreezerActivate))
	js.Global().Set("FreezerDeactivate", js.FuncOf(ctx.FreezerDeactivate))
	js.Global().Set("HotHoldActivate", js.FuncOf(ctx.HotHoldActivate))
	js.Global().Set("HotHoldDeactivate", js.FuncOf(ctx.HotHoldDeactivate))
	js.Global().Set("FreezerUnlock", js.FuncOf(ctx.FreezerUnlock))
	js.Global().Set("TransportPosSetParams", js.FuncOf(ctx.TransportPosSetParams))
	js.Global().Set("ChangeOpMode", js.FuncOf(ctx.ChangeOpMode))

	js.Global().Set("ClearLog", js.FuncOf(ctx.ClearLog))
	js.Global().Set("ClearPidLog", js.FuncOf(ctx.ClearPidLog))

	js.Global().Set("PidSetParams", js.FuncOf(ctx.PidSetParams))

	js.Global().Set("GetState", js.FuncOf(ctx.GetState))
	js.Global().Set("DrawerLock", js.FuncOf(ctx.DrawerLock))
	js.Global().Set("DrawerUnlock", js.FuncOf(ctx.DrawerUnlock))

	js.Global().Set("TemperatureControlSetParams", js.FuncOf(ctx.TemperatureControlSetParams))
	js.Global().Set("TemperatureControlEnable", js.FuncOf(ctx.TemperatureControlEnable))
	js.Global().Set("TemperatureControlDisable", js.FuncOf(ctx.TemperatureControlDisable))

	js.Global().Set("AgitatorRun", js.FuncOf(ctx.AgitatorRun))
	js.Global().Set("AgitatorStop", js.FuncOf(ctx.AgitatorStop))

	js.Global().Set("IngredientSetParams", js.FuncOf(ctx.IngredientSetParams))

	js.Global().Set("TransportMove", js.FuncOf(ctx.TransportMove))
	js.Global().Set("StopAllSteppers", js.FuncOf(ctx.StopAllSteppers))

}

func main() {

	c := make(chan struct{}, 0)

	// Init
	ctx := Ctx{}
	ctx.wsConn = false

	ctx.registerCallbacks()
	pidAreaDefaultValue := "Run" + "\t" + "Loop" + "\t" + "t" + "\t" + "Sp" + "\t" + "Cv" + "\t" + "Err" + "\t" + "Int" + "\t" + "Der" + "\t" + "P" + "\t" + "I" + "\t" + "D" + "\t" + "Pv\n"
	ctx.getElementByID("txtPidAreaTitle").Set("value", pidAreaDefaultValue)

	fmt.Println("WASM Go Initialized")
	<-c
}
