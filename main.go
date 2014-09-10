package hello

import (
//    "fmt"
    "net/http"
    "io/ioutil"
    "encoding/xml"
    "html/template"
    "net/url"

    "appengine"
    "appengine/urlfetch"
)

type Weather struct {
        XMLName string `xml:"data"`
        Weather struct {
            XMLName string `xml:"weather"`
            TempMaxC string `xml:"tempMaxC"`
            TempMinC string `xml:"tempMinC"`
        }
        Location struct {
            XMLName string `xml:"request"`
            Current string `xml:"query"`
        }
        Error struct {
            XMLName string `xml:"error"`
            Msg string `xml:"msg"`
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
        getWeather(w, r)
    }
}

func getWeather(w http.ResponseWriter, r *http.Request) {
    appengineContext := appengine.NewContext(r)
    client := urlfetch.Client(appengineContext)

    r.ParseForm()
    location := []string{
        r.Form["location1"][0], 
        r.Form["location2"][0], 
        r.Form["location3"][0], 
        r.Form["location4"][0],
        r.Form["location5"][0],
    }

    weatherc := make(chan Weather)
    errc := make(chan error)

    for i:=0; i<5; i++ {
        go func (location string) {
            if location == "" {
                weatherc <- Weather{}
            } else {
                weather, err := fetchWeather(client, url.QueryEscape(location))
                if err != nil {
                    errc <- err
                    return
                } else {
                    weatherc <- weather
                }
            }
        }(location[i])
    }

    locationWeather := make([]Weather, 5)

    for i:=0; i<5; i++ {
        select {
            case weather := <- weatherc:
                locationWeather[i] = weather
            case err := <- errc:
                if err != nil {
                    t, _ := template.ParseFiles("templates/error.tmpl")
                    t.Execute(w, nil)
                    //http.Error(w, err.Error(), http.StatusInternalServerError)
                    return
                }
        }
    }

    t, err := template.ParseFiles("templates/weather.tmpl")
    if err != nil {
        t, _ := template.ParseFiles("templates/error.tmpl")
        t.Execute(w, nil)
        return
    }

    t.Execute(w, locationWeather)
//    fmt.Fprintf(w, "%s %s %s %s\n", locationWeather[0].Location, locationWeather[1].Location, locationWeather[2].Location, locationWeather[3].Location)
}

func fetchWeather(client *http.Client, location string) (Weather, error) {
    resp, err := client.Get("http://api.worldweatheronline.com/free/v1/weather.ashx?key=b07b0d3aafb4b1229d8d297c1c64d61637f6c5f4&q=" + location + "&format=xml&date=today")
    if err != nil {
        return Weather{}, err
    }

    contents, err := ioutil.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
        return Weather{}, err
    }

    var weather Weather
    err = xml.Unmarshal(contents, &weather)
    if err != nil {
        return Weather{}, err
    }

    return weather, nil
}