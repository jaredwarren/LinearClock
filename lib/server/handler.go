package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jaredwarren/clock/lib/config"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	c, err := config.ReadConfig("config.gob")
	if err != nil {
		if errorIsMissing(err) {
			c = config.DefaultConfig
		} else {
			fmt.Fprintf(w, "get config error:%+v", err)
			return
		}
	}

	files := []string{
		"templates/home.html",
		"templates/layout.html",
	}
	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"ColorString": ColorString,
		"TimeNum":     TimeNum,
	}).ParseFiles(files...)
	if err != nil {
		fmt.Fprintf(w, "parse template error:%+v", err)
		return
	}
	err = tmpl.Execute(w, c)
	if err != nil {
		fmt.Fprintf(w, "exec temp error:%+v", err)
		return
	}
}

func ColorString(color uint32) string {
	return fmt.Sprintf("#%06X", color)
}

func TimeNum(t time.Duration) string {
	return fmt.Sprintf("%d", int(t.Seconds()))
}

func TimeFormat(t time.Time) string {
	return t.Format("2006-01-02T15:04")
}

func errorIsMissing(err error) bool {
	return strings.Contains(err.Error(), "no such file or directory")
}

func (s *Server) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "Error parsing form data: %v", err)
		return
	}

	// Get Old config if available
	c, err := config.ReadConfig("config.gob")
	if err != nil {
		if errorIsMissing(err) {
			c = config.DefaultConfig
		} else {
			fmt.Fprintf(w, "get config error:%+v", err)
			return
		}
	}

	fmt.Println("Brightness:" + r.FormValue("brightness"))
	{
		i, err := strconv.Atoi(r.FormValue("brightness"))
		if err != nil {
			fmt.Fprintf(w, "convert 'brightness' error (%+v):%+v", i, err)
			return
		}
		c.Brightness = i
	}

	fmt.Println("RefreshRate:" + r.FormValue("refresh-rate"))
	{
		i, err := strconv.ParseInt(r.FormValue("refresh-rate"), 10, 64)
		if err != nil {
			fmt.Fprintf(w, "convert 'refresh-rate' error (%+v):%+v", i, err)
			return
		}
		if i < 1 || i > 900 {
			fmt.Fprintf(w, "invalid 'refresh-rate' error (%+v)", i)
			return
		}
		c.RefreshRate = time.Second * time.Duration(i)
	}

	fmt.Println("Gap:" + r.FormValue("gap"))
	{
		i, err := strconv.Atoi(r.FormValue("gap"))
		if err != nil {
			fmt.Fprintf(w, "convert 'gap' error (%+v):%+v", i, err)
			return
		}
		c.Gap = i
	}

	fmt.Println("Tick.StartLed:" + r.FormValue("tick.start-led"))
	{
		i, err := strconv.Atoi(r.FormValue("tick.start-led"))
		if err != nil {
			fmt.Fprintf(w, "convert 'tick.start-led' error (%+v):%+v", i, err)
			return
		}
		c.Tick.StartLed = i
	}

	fmt.Println("Tick.TicksPerHour:" + r.FormValue("tick.ticks-per-hour"))
	{
		i, err := strconv.Atoi(r.FormValue("tick.ticks-per-hour"))
		if err != nil {
			fmt.Fprintf(w, "convert 'tick.ticks-per-hour' error (%+v):%+v", i, err)
			return
		}
		c.Tick.TicksPerHour = i
	}

	fmt.Println("Tick.NumHours:" + r.FormValue("tick.num-hours"))
	{
		i, err := strconv.Atoi(r.FormValue("tick.num-hours"))
		if err != nil {
			fmt.Fprintf(w, "convert 'tick.num-hours' error (%+v):%+v", i, err)
			return
		}
		c.Tick.NumHours = i
	}

	fmt.Println("Tick.StartHour:" + r.FormValue("tick.start-hour"))
	{
		i, err := strconv.Atoi(r.FormValue("tick.start-hour"))
		if err != nil {
			fmt.Fprintf(w, "convert 'tick.start-hour' error (%+v):%+v", i, err)
			return
		}
		c.Tick.StartHour = i
	}

	fmt.Println("Tick.PastColor:" + r.FormValue("tick.past-color"))
	{
		v := r.FormValue("tick.past-color")
		color, err := hexStringToUint32(v)
		if err != nil {
			fmt.Fprintf(w, "convert 'tick.past-color' error (%s):%+v", v, err)
			return
		}
		c.Tick.PastColor = color
	}

	fmt.Println("Tick.PresentColor:" + r.FormValue("tick.present-color"))
	{
		v := r.FormValue("tick.present-color")
		color, err := hexStringToUint32(v)
		if err != nil {
			fmt.Fprintf(w, "convert 'tick.present-color' error (%s):%+v", v, err)
			return
		}
		c.Tick.PresentColor = color
	}

	fmt.Println("Tick.FutureColor:" + r.FormValue("tick.future-color"))
	{
		v := r.FormValue("tick.future-color")
		color, err := hexStringToUint32(v)
		if err != nil {
			fmt.Fprintf(w, "convert 'tick.future-color' error (%s):%+v", v, err)
			return
		}
		c.Tick.FutureColor = color
	}

	fmt.Println("Num.PastColor:" + r.FormValue("num.past-color"))
	{
		v := r.FormValue("num.past-color")
		color, err := hexStringToUint32(v)
		if err != nil {
			fmt.Fprintf(w, "convert 'num.past-color' error (%s):%+v", v, err)
			return
		}
		c.Num.PastColor = color
	}

	fmt.Println("Num.PresentColor:" + r.FormValue("num.present-color"))
	{
		v := r.FormValue("num.present-color")
		color, err := hexStringToUint32(v)
		if err != nil {
			fmt.Fprintf(w, "convert 'num.present-color' error (%s):%+v", v, err)
			return
		}
		c.Num.PresentColor = color
	}

	fmt.Println("Num.FutureColor:" + r.FormValue("num.future-color"))
	{
		v := r.FormValue("num.future-color")
		color, err := hexStringToUint32(v)
		if err != nil {
			fmt.Fprintf(w, "convert 'num.future-color' error (%s):%+v", v, err)
			return
		}
		c.Num.FutureColor = color
	}

	// Write Config
	err = config.WriteConfig("config.gob", c)
	if err != nil {
		fmt.Fprintf(w, "write config error :%+v", err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func hexStringToUint32(hexStr string) (uint32, error) {
	hexStr = strings.TrimPrefix(hexStr, "#")

	r, err := strconv.ParseUint(hexStr[0:2], 16, 8)
	if err != nil {
		return 0, fmt.Errorf("parse color x in r - %w", err)
	}
	g, err := strconv.ParseUint(hexStr[2:4], 16, 8)
	if err != nil {
		return 0, fmt.Errorf("parse color x in g - %w", err)
	}
	b, err := strconv.ParseUint(hexStr[4:6], 16, 8)
	if err != nil {
		return 0, fmt.Errorf("parse color x in b - %w", err)
	}

	return uint32(r)<<16 | uint32(g)<<8 | uint32(b), nil
}
