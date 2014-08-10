package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"runtime"
)

var (
	ErrorBadIdent = errors.New("Unable to parse id")
)

type Post struct {
	ID   bson.ObjectId `json:"id" bson:"_id"`
	Text string        `json:"text" bson:"text"`
}

type DataBase struct {
	session *mgo.Session
	db      *mgo.Database
	posts   *mgo.Collection
}

func ParseID(s string) (id bson.ObjectId, err error) {
	if !bson.IsObjectIdHex(s) {
		return id, ErrorBadIdent
	}
	return bson.ObjectIdHex(s), nil
}

func (db *DataBase) AddPost(post Post) error {
	return db.posts.Insert(post)
}

func (d *DataBase) GetAllPosts() (posts []Post, err error) {
	return posts, d.posts.Find(nil).All(&posts)
}

func (d *DataBase) GetPost(id bson.ObjectId) (*Post, error) {
	query := bson.M{"_id": id}
	post := &Post{}
	return post, d.posts.Find(query).One(post)
}

const (
	postsCollection = "posts"
)

func GetDB(name, url string) (*DataBase, error) {
	db := DataBase{}
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}

	db.session = session
	db.db = session.DB(name)
	db.posts = db.db.C(postsCollection)

	return &db, nil
}

type Application struct {
	DB  *DataBase
	ids chan int64
}

func ShowErr(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func (a *Application) StartIDDispatcher() {
	a.ids = make(chan int64, 100)
	var id int64
	log.Println("dispatcher started")
	for {
		id += 1
		a.ids <- id
	}
}

func (a *Application) GetID() string {
	log.Println("dispatching")
	id := <-a.ids
	return fmt.Sprintf("%x", id)
}

func (a *Application) GetPost(c *gin.Context) {
	idStr := c.Params.ByName("id")
	id, err := ParseID(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ShowErr(err))
		return
	}
	post, err := a.DB.GetPost(id)
	if err == mgo.ErrNotFound {
		c.JSON(http.StatusNotFound, ShowErr(err))
		return
	}
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, ShowErr(err))
		return
	}
	a.Render(c, post)
}

func (a *Application) GetAllPosts(c *gin.Context) {
	posts, err := a.DB.GetAllPosts()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, ShowErr(err))
		return
	}
	a.Render(c, posts)
}

func (a *Application) Render(c *gin.Context, val interface{}) {
	c.Writer.WriteHeader(200)
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(val); err != nil {
		c.ErrorTyped(err, gin.ErrorTypeInternal, val)
		c.Abort(http.StatusInternalServerError)
	}
}

func (a *Application) AddPost(c *gin.Context) {
	var post Post
	decoder := json.NewDecoder(c.Req.Body)
	if err := decoder.Decode(&post); err != nil {
		c.JSON(http.StatusBadRequest, ShowErr(err))
		return
	}
	post.ID = bson.NewObjectId()
	if err := a.DB.AddPost(post); err != nil {
		c.JSON(http.StatusInternalServerError, ShowErr(err))
		return
	}
	a.Render(c, "ok")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	db, err := GetDB("gost-dev", "localhost")
	if err != nil {
		log.Fatal(err)
	}
	app := &Application{}
	app.DB = db
	go app.StartIDDispatcher()
	id := app.GetID()
	log.Println(id)
	r := gin.Default()
	r.GET("/post", app.GetAllPosts)
	r.PUT("/post", app.AddPost)
	r.GET("/post/:id/", app.GetPost)
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	r.Static("/static", "static")
	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
