package hello

import (
//    "fmt"
    "net/http"
    "io/ioutil"
    "encoding/json"
    "html/template"
    "errors"

    "appengine"
    "appengine/urlfetch"
)

type WeatherData struct {
    TempMaxC string
    TempMinC string
}

type LocationData struct {
    Query string
}

type ResponseData struct {
    Location string
    Weather WeatherData
}

type Weather struct {
    Data struct {
        Weather []WeatherData
        Request []LocationData
        Error []map[string]string
    }
}

func init() {
    var chttp = http.NewServeMux()
    chttp.Handle("/", http.FileServer(http.Dir("./")))
    http.HandleFunc("/", homeHandler)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
        t, _ := template.ParseFiles("templates/input.tmpl")
        t.Execute(w, nil)
    } else if r.Method == "POST" {
        getWeatherHandler(w, r)
    }
}

func getWeatherHandler(w http.ResponseWriter, r *http.Request) {
    appengineContext := appengine.NewContext(r)
    client := urlfetch.Client(appengineContext)

    r.ParseForm()
    location := []string{
        r.Form["location1"][0], 
        r.Form["location2"][0], 
        r.Form["location3"][0], 
        r.Form["location4"][0],
    }

    weatherc := make(chan ResponseData)
    errc := make(chan error)

    for i:=0; i<4; i++ {
        go func (location string) {
            if location == "" {
                weatherc <- ResponseData{Location: "", Weather: WeatherData{}}
            } else {
                responseData, err := fetchResponseData(client, location)
                if err != nil {
                    errc <- err
                    return
                } else {
                    weatherc <- responseData
                }
            }
        }(location[i])
    }

    locationWeather := make([]ResponseData, 4)

    for i:=0; i<4; i++ {
        select {
            case weather := <- weatherc:
                locationWeather[i] = weather
            case err := <- errc:
                if err != nil {
                    http.Error(w, err.Error(), http.StatusInternalServerError)
                    return
                }
        }
    }

    t, err := template.ParseFiles("templates/weather.tmpl")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    t.Execute(w, locationWeather)
//    fmt.Fprintf(w, "%s %s %s %s\n", locationWeather[0].Location, locationWeather[1].Location, locationWeather[2].Location, locationWeather[3].Location)
}

func fetchResponseData(client *http.Client, location string) (ResponseData, error) {
    resp, err := client.Get("http://api.worldweatheronline.com/free/v1/weather.ashx?key=b07b0d3aafb4b1229d8d297c1c64d61637f6c5f4&q=" + location + "&format=json&date=today")
    if err != nil {
        return ResponseData{}, err
    }

    contents, err := ioutil.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
        return ResponseData{}, err
    }

    var weather Weather
    err = json.Unmarshal(contents, &weather)
    if err != nil {
        return ResponseData{}, err
    }

    if len(weather.Data.Error) > 0 {
        return ResponseData{}, errors.New(weather.Data.Error[0]["msg"])
    }

    responseData := ResponseData{Location: weather.Data.Request[0].Query, Weather: weather.Data.Weather[0]}

    return responseData, nil
}