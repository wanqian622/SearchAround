package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"log"
	"strconv"
)

const (
	DISTANCE = "200km"
)



type Location struct{
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Post struct{
	User string `json:"user"`
	Message string `json:"message"`
	Location Location `json:"location"`
}
/*
auto to map
{
	user:"jack",
	message:"hello",
	location:{
		lat:37,
		lon:-118
	}
}
 */

func main() {
	fmt.Println("start-service")
	http.HandleFunc("/post",handlerPost) //if /post, we call handlerPost
	http.HandleFunc("/search",handlerSearch) //if /search?query, we call handlerSearch
	log.Fatal(http.ListenAndServe(":8080",nil)) // listen 8080 port and start server 所有的东西好了在启动server
}


// search nearby posts
func handlerSearch(w http.ResponseWriter, r *http.Request){
	fmt.Println("Received a search request")
	lat := r.URL.Query().Get("lat")
	lt,_ := strconv.ParseFloat(lat,64)  // convert string to float64 type which we defined
	lon := r.URL.Query().Get("lon")
	ln,_ := strconv.ParseFloat(lon,64)  // _ represent error, and we ensure it won't happen error, we do not use it, sp we use _

	// build a new Post object
	// & like new keyword

	ran := DISTANCE
	if val := r.URL.Query().Get("range"); val != ""{
		ran = val + "km"
	}

	fmt.Fprintf(w,"range is %s\n", ran)

	p := &Post{
		User:"1111",
		Message:"This is whatever",
		Location:Location{
			Lat:lt,
			Lon:ln,
		},
	}

	js, err := json.Marshal(p)  // convert to json format
	if err != nil{
		return
	}

	w.Header().Set("Content-Type","application-json")
	w.Write(js)  // return to client, write to response
	fmt.Fprintf(w,"Lat is %s Lon is %s\n", lat, lon)
}

// http handler, handle Post request
func handlerPost(w http.ResponseWriter, r *http.Request){
	fmt.Println("Received a request for post")
	decoder :=json.NewDecoder(r.Body)
	var p Post
	if err := decoder.Decode(&p); err != nil{
		panic(err) // output err
		return
	}

	fmt.Fprintf(w, "Post is received %s\n", p.Message) // response to client, we write things to client(response)  "Post is received 一生要去的地方"

}

