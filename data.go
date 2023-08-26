package main

import (
	"context"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/ini.v1"
)

// MongoUsers for get data from mongodb
type MongoUsers struct {
	ID       primitive.ObjectID `json:"id" bson:"_id"`
	UserID   int32              `json:"userId" bson:"userId"`
	Name     string             `json:"name" bson:"name"`
	PassWord string             `json:"password" bson:"password"`
}

// MongoEntries for get data from mongodb
type MongoEntries struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	EntryID     int32              `json:"entryId" bson:"entryId"`
	EntryCode   string             `json:"entryCode" bson:"entryCode"`
	PublishDate string             `json:"publishDate" bson:"publishDate"`
	Title       string             `json:"title" bson:"title"`
	Content     string             `json:"content" bson:"content"`
	Tag         []string           `json:"tag" bson:"tag"`
	IsPublished int32              `json:"isPublished" bson:"isPublished"`
	AuthorID    int32              `json:"authorId" bson:"authorId"`
	CreatedAt   string             `json:"createdAt" bson:"createdAt"`
	UpdatedAt   string             `json:"updatedAt" bson:"updatedAt"`
}

// EntryItem for view
type EntryItem struct {
	EntryID     int
	URI         string
	PublishDate string
	Title       string
	Content     string
	Tags        []TagItem
}

// TagItem for view
type TagItem struct {
	TagName string
	TagURI  string
	Count   int
}

// TitleList for tag search
type TitleList struct {
	URI         string
	PublishDate string
	Title       string
	Tags        []TagItem
}

// Settings struct
type Settings struct {
	HttpdPort     string
	BlogURL       string
	RootPath      string
	BackendURI    string
	PagePerView   int
	SessionName   string
	LoggedinKey   string
	LoggedinValue string
	DBUser        string
	DBPassword    string
	DBName        string
	DBHost        string
	DBPort        string
}

// Paginator struct
type Paginator struct {
	IsExists bool
	URI      string
}

// CacheEntries struct
type CacheEntries struct {
	EntryItems        []EntryItem
	NextPaginator     Paginator
	PreviousPaginator Paginator
}

// 定数
const (
	IsPublished      = 1
	MoreLinkString   = "<!--more-->"
	SettingsFilePath = "./settings.ini"
)

// fields
var (
	// settings
	settings Settings
	// db context
	ctx context.Context
	// db object
	client *mongo.Client
	// pagenate URL prefix
	paginatorPrefixURI string
	tagPrefixURI       string
	// cache tags
	cacheTagsAll []TagItem
	// cache entry (string = entryCode)
	cacheEntry map[string]EntryItem
	// cache entries for page (int = page)
	cacheEntriesForPage map[int]CacheEntries
	// cache titleList for tag page (string = tagName)
	cacheTitleList map[string][]TitleList
)

func initializeData() {
	// settings
	iniFile, err := ini.Load(SettingsFilePath)
	if err != nil {
		panic("ini load error")
	}
	settings = Settings{
		HttpdPort:     iniFile.Section("app").Key("HttpdPort").String(),
		BlogURL:       iniFile.Section("site").Key("BlogURL").String(),
		RootPath:      iniFile.Section("site").Key("RootPath").String(),
		BackendURI:    iniFile.Section("site").Key("BackendURI").String(),
		PagePerView:   iniFile.Section("site").Key("PagePerView").MustInt(),
		SessionName:   iniFile.Section("site").Key("SessionName").String(),
		LoggedinKey:   iniFile.Section("site").Key("LoggedinKey").String(),
		LoggedinValue: iniFile.Section("site").Key("LoggedinValue").String(),
		DBUser:        iniFile.Section("db").Key("DBUser").String(),
		DBPassword:    iniFile.Section("db").Key("DBPassword").String(),
		DBName:        iniFile.Section("db").Key("DBName").String(),
		DBHost:        iniFile.Section("db").Key("DBHost").String(),
		DBPort:        iniFile.Section("db").Key("DBPort").String(),
	}
	// link urls
	paginatorPrefixURI = settings.RootPath + "page/"
	tagPrefixURI = settings.RootPath + "tag/"
	// init DB
	ctx = context.Background()
	credential := options.Credential{
		AuthSource: settings.DBName,
		Username:   settings.DBUser,
		Password:   settings.DBPassword,
	}
	uri := "mongodb://" + settings.DBHost + ":" + settings.DBPort
	temp, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetAuth(credential))
	if err != nil {
		panic("db connect error")
	}
	client = temp
	// init tag slice
	getTagsAll()
	// cache map init
	cacheEntry = make(map[string]EntryItem)
	cacheEntriesForPage = make(map[int]CacheEntries)
	cacheTitleList = make(map[string][]TitleList)
}

func closeConnection() {
	client.Disconnect(ctx)
}

// tag エントリから全てのカテゴリを抽出する(重複は無視)
func getTagsAll() {
	cacheTagsAll = nil
	entries := client.Database(settings.DBName).Collection("entries")
	cur, err := entries.Find(ctx, bson.D{})
	if err != nil {
		return
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var result MongoEntries
		err := cur.Decode(&result)
		if err != nil {
			//log.Fatal(err)
			continue
		}
		for _, name := range result.Tag {
			// goにはin_array, List<T>.Containsみたいなものは無いみたいなので自前チェック
			isExists := false
			for idx := 0; idx < len(cacheTagsAll); idx++ {
				if cacheTagsAll[idx].TagName == name {
					isExists = true
					cacheTagsAll[idx].Count++
					break
				}
			}
			if !isExists {
				cacheTagsAll = append(cacheTagsAll, TagItem{
					TagName: name,
					TagURI:  tagPrefixURI + name,
					Count:   1,
				})
			}
		}
	}
}

// get entry item with paginator flag(next, previous)
func getEntryList(page int) ([]EntryItem, Paginator, Paginator) {
	// cache exists check & return
	if val, ok := cacheEntriesForPage[page]; ok {
		return val.EntryItems, val.NextPaginator, val.PreviousPaginator
	}
	// get entry list
	nextPaginator := Paginator{}
	if page > 0 {
		// pageから-1にして0の場合はindexにする
		if (page - 1) == 0 {
			nextPaginator = Paginator{IsExists: true, URI: settings.RootPath}
		} else {
			nextPaginator = Paginator{IsExists: true, URI: paginatorPrefixURI + strconv.Itoa(page-1)}
		}
	}
	previousPaginator := Paginator{}
	offset := page * settings.PagePerView
	var entryItems []EntryItem
	entries := client.Database(settings.DBName).Collection("entries")
	// paginateのために1件多く取得する
	findOption := options.Find().SetSort(bson.D{{Key: "publishDate", Value: -1}}).SetSkip(int64(offset)).SetLimit(int64(settings.PagePerView + 1))
	cur, err := entries.Find(ctx, bson.D{{Key: "isPublished", Value: 1}}, findOption)
	if err != nil {
		//log.Fatal(err)
		return entryItems, nextPaginator, previousPaginator
	}
	// findはスライス等で返ってこない - *Cursol型で返ってくるので下記のようにループ回して取得
	// PagePerViewの値を超えて存在した場合、Paginater->Previousは有効になる
	defer cur.Close(ctx)
	index := 0
	for cur.Next(ctx) {
		if index >= settings.PagePerView {
			previousPaginator = Paginator{IsExists: true, URI: paginatorPrefixURI + strconv.Itoa(page+1)}
			break
		}
		var result MongoEntries
		err := cur.Decode(&result)
		if err != nil {
			//log.Fatal(err)
			return entryItems, nextPaginator, previousPaginator
		}
		var tags []TagItem
		for _, v := range result.Tag {
			tags = append(tags, TagItem{TagName: v, TagURI: tagPrefixURI + v})
		}
		entryItems = append(entryItems, EntryItem{
			EntryID:     int(result.EntryID),
			URI:         settings.RootPath + result.EntryCode,
			PublishDate: result.PublishDate,
			Title:       result.Title,
			Content:     result.Content,
			Tags:        tags,
		})
		index++
	}
	// save cache
	cacheEntriesForPage[page] = CacheEntries{EntryItems: entryItems, NextPaginator: nextPaginator, PreviousPaginator: previousPaginator}
	return entryItems, nextPaginator, previousPaginator
}

func getEntry(entryCode string) EntryItem {
	// cache exists check & return
	if val, ok := cacheEntry[entryCode]; ok {
		return val
	}
	// get entry
	var entryItem EntryItem
	entries := client.Database(settings.DBName).Collection("entries")
	var result MongoEntries
	err := entries.FindOne(ctx, bson.D{{Key: "entryCode", Value: entryCode}, {Key: "isPublished", Value: 1}}).Decode(&result)
	if err != nil {
		return entryItem
	}
	var tags []TagItem
	for _, v := range result.Tag {
		tags = append(tags, TagItem{TagName: v, TagURI: tagPrefixURI + v})
	}
	entryItem = EntryItem{
		EntryID:     int(result.EntryID),
		URI:         settings.RootPath + result.EntryCode,
		PublishDate: result.PublishDate,
		Title:       result.Title,
		Content:     result.Content,
		Tags:        tags,
	}
	// save cache
	cacheEntry[entryCode] = entryItem
	return entryItem
}

func getTitleList(tagName string) []TitleList {
	// cache exists check & return
	if val, ok := cacheTitleList[tagName]; ok {
		return val
	}
	// get title list
	var titleList []TitleList
	entries := client.Database(settings.DBName).Collection("entries")
	findOption := options.Find().SetSort(bson.D{{Key: "publishDate", Value: -1}})
	cur, err := entries.Find(ctx, bson.D{{Key: "tag", Value: tagName}, {Key: "isPublished", Value: 1}}, findOption)
	if err != nil {
		//log.Fatal(err)
		return titleList
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var result MongoEntries
		err := cur.Decode(&result)
		if err != nil {
			//log.Fatal(err)
			return titleList
		}
		var tags []TagItem
		for _, v := range result.Tag {
			tags = append(tags, TagItem{TagName: v, TagURI: tagPrefixURI + v})
		}
		titleList = append(titleList, TitleList{
			URI:         settings.RootPath + result.EntryCode,
			PublishDate: result.PublishDate,
			Title:       result.Title,
			Tags:        tags,
		})
	}
	// save cache
	cacheTitleList[tagName] = titleList
	return titleList
}

func getAllEntries() []MongoEntries {
	var allEntries []MongoEntries
	entries := client.Database(settings.DBName).Collection("entries")
	findOption := options.Find().SetSort(bson.D{{Key: "publishDate", Value: -1}})
	cur, err := entries.Find(ctx, bson.D{}, findOption)
	if err != nil {
		return allEntries
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var result MongoEntries
		err := cur.Decode(&result)
		if err != nil {
			//log.Fatal(err)
			return allEntries
		}
		allEntries = append(allEntries, result)
	}
	return allEntries
}

func getUser(name string) MongoUsers {
	var user MongoUsers
	users := client.Database(settings.DBName).Collection("users")
	findOption := options.Find().SetLimit(1)
	cur, err := users.Find(ctx, bson.D{{Key: "name", Value: name}}, findOption)
	if err != nil {
		// logging!!
		return user
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		err := cur.Decode(&user)
		if err != nil {
			//log.Fatal(err)
			return user
		}
	}
	return user
}
