package main

import (
	"context"
	"cloud.google.com/go/storage"
	elastic "gopkg.in/olivere/elastic.v3"
	"fmt"
	"net/http"
	"encoding/json"
	"log"
	"strconv"
	"reflect"
	"github.com/pborman/uuid"
	"io"
)

const (
	INDEX = "around"
	TYPE = "post"
	DISTANCE = "200km"
	// Needs to update
	//PROJECT_ID = "around-xxx"
	//BT_INSTANCE = "around-post"
	// Needs to update this URL if you deploy it to cloud.
	ES_URL = "http://35.190.175.79:9200"
	BUCKET_NAME = "post-images-203720"
)



type Location struct{
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Post struct{
	User string `json:"user"`
	Message string `json:"message"`
	Location Location `json:"location"`
	Url    string `json:"url"`
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
	//to recognize es's data structure
	//convert normal type(like Location) to geo-index which can be recognized by es
	// Create a client
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(INDEX).Do()
	if err != nil {
		panic(err)
	}
	if !exists {
		// Create a new index.
		mapping := `{
                    "mappings":{
                           "post":{
                                  "properties":{
                                         "location":{
                                                "type":"geo_point"
                                         }
                                  }
                           }
                    }
             }
             `
		// create INDEX which is used in our service
		_, err := client.CreateIndex(INDEX).Body(mapping).Do()
		if err != nil {
			// Handle error
			panic(err)
		}
	}


	fmt.Println("start-service")
	http.HandleFunc("/post",handlerPost) //if /post, we call handlerPost
	http.HandleFunc("/search",handlerSearch) //if /search?query, we call handlerSearch
	log.Fatal(http.ListenAndServe(":8080",nil)) // listen 8080 port and start server 所有的东西好了在启动server
}


// search nearby posts
func handlerSearch(w http.ResponseWriter, r *http.Request){

	fmt.Println("Received one request for search")
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64) // convert string to float64 type which we defined
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)

	// range is optional
	ran := DISTANCE
	if val := r.URL.Query().Get("range"); val != "" {
		ran = val + "km"
	}


	fmt.Printf( "Search received: %f %f %s\n", lat, lon, ran)

	// Create a client
	//create a connection to ES. If there is err, return.
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	//Prepare a geo based query to find posts within a geo box
	// Define geo distance query as specified in
	// https://www.elastic.co/guide/en/elasticsearch/reference/5.2/query-dsl-geo-distance-query.html
	q := elastic.NewGeoDistanceQuery("location")
	q = q.Distance(ran).Lat(lat).Lon(lon)

	// Some delay may range from seconds to minutes. So if you don't get enough results. Try it later.
	//Get the results based on Index (similar to dataset) and query (q that we just prepared). Pretty means to format the output
	searchResult, err := client.Search().
		Index(INDEX).
		Query(q).
		Pretty(true).
		Do()
	if err != nil {
		// Handle error
		panic(err)
	}

	// searchResult is of type SearchResult and returns hits, suggestions,
	// and all kinds of other information from Elasticsearch.
	fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)
	// TotalHits is another convenience function that works even when something goes wrong.
	fmt.Printf("Found a total of %d post\n", searchResult.TotalHits())

	// Each is a convenience function that iterates over hits in a search result.
	// It makes sure you don't need to check for nil values in the response.
	// However, it ignores errors in serialization.
	// the result returned from ElasticSearch converts to Post type
	//Iterate the result results and if they are type of Post (typ)
	//Cast an item to Post, equals to p = (Post) item in java
	//Add the p to an array, equals ps.add(p) in java

	var typ Post
	var ps []Post   // post type array
	for _, item := range searchResult.Each(reflect.TypeOf(typ)) { // instance of
		p := item.(Post) // p = (Post) item
		fmt.Printf("Post by %s: %s at lat %v and lon %v\n", p.User, p.Message, p.Location.Lat, p.Location.Lon)
		// TODO(student homework): Perform filtering based on keywords such as web spam etc.
		ps = append(ps, p)

	}
	//Convert the go object to a string
	js, err := json.Marshal(ps)
	if err != nil {
		panic(err)
		return
	}
	//Allow cross domain visit for javascript.
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(js)

}

// http handler, handle Post request
func handlerPost(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
	// 32 << 20 is the maxMemory param for ParseMultipartForm, equals to 32MB (1MB = 1024 * 1024 bytes = 2^20 bytes)
	// After you call ParseMultipartForm, the file will be saved in the server memory with maxMemory size.
	// If the file size is larger than maxMemory, the rest of the data will be saved in a system temporary file.
	r.ParseMultipartForm(32 << 20)

	// Parse from form data.
	fmt.Printf("Received one post request %s\n", r.FormValue("message"))
	lat, _ := strconv.ParseFloat(r.FormValue("lat"), 64)
	lon, _ := strconv.ParseFloat(r.FormValue("lon"), 64)
	p := &Post{
		User:    "1111",
		Message: r.FormValue("message"),
		Location: Location{
			Lat: lat,
			Lon: lon,
		},
	}



	//to produce unique id
	id := uuid.New()

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image is not available", http.StatusInternalServerError)
		fmt.Printf("Image is not available %v.\n", err)
		return
	}
	defer file.Close()

	ctx := context.Background()  // context like build a session to get credital, to link with storage

	// replace it with your real bucket name.
	_, attrs, err := saveToGCS(ctx, file, BUCKET_NAME, id)
	if err != nil {
		http.Error(w, "GCS is not setup", http.StatusInternalServerError)
		fmt.Printf("GCS is not setup %v\n", err)
		return
	}

	// Update the media link after saving to GCS.
	p.Url = attrs.MediaLink

	// Save to ES.
	saveToES(p, id)

	// Save to BigTable.
	//saveToBigTable(p, id)


}


func saveToGCS(ctx context.Context, r io.Reader, bucketName, name string) (*storage.ObjectHandle, *storage.ObjectAttrs, error) {
	// Student questions
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	defer client.Close(); // when this funtion is finished, we execute it, 逆向执行

	bucket := client.Bucket(bucketName)
	// check if the bucket exists
	if _, err = bucket.Attrs(ctx); err != nil{
		return nil, nil, err
	}

	// build an object to copy this file to storage
	obj := bucket.Object(name)  // bucket 相当于一个folder, object相当于folder(bucket)里面的一个文件
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil{
		return nil, nil, err
	}

	if err := w.Close(); err != nil {
		return nil, nil, err
	}

	// 修改访问权限,让所有用户可读
	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return nil, nil, err
	}

	// 获取传上去的文件的url
	attrs, err := obj.Attrs(ctx)
	fmt.Printf("Post is saved to GCS: %s\n", attrs.MediaLink)
	return obj, attrs, err
}


// Save a post to ElasticSearch
func saveToES(p *Post, id string) {
	// Create a client
	es_client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	// Save it to index
	_, err = es_client.Index().
		Index(INDEX).
		Type(TYPE).
		Id(id).
		BodyJson(p).
		Refresh(true).
		Do()
	if err != nil {
		panic(err)
		return
	}

	fmt.Printf("Post is saved to Index: %s\n", p.Message)
}


