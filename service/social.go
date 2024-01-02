package service

import (
	"mysrv/util"
	"time"
	"fmt"
)

var socialSQLTables = `

CREATE TABLE IF NOT EXISTS social_post (
	postID INTEGER PRIMARY KEY AUTOINCREMENT,
	posterID INTEGER NOT NULL,
	postText TEXT NOT NULL,
	postTime DATETIME NOT NULL,
	commentCount INTEGER NOT NULL DEFAULT 0,
	likeCount INTEGER NOT NULL DEFAULT 0,
	dislikeCount INTEGER NOT NULL DEFAULT 0,
	CHECK(likeCount >= 0),
	CHECK(dislikeCount >= 0),
	FOREIGN KEY(posterID) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS social_comment (
	commentID INTEGER PRIMARY KEY AUTOINCREMENT,
	commenterID INTEGER NOT NULL,
	commentText TEXT NOT NULL,
	commentTime DATETIME NOT NULL,
	postID UNSIGNED INTEGER NOT NULL,
	parentCommentID INTEGER,
	childrenCount INTEGER NOT NULL DEFAULT 0,
	likeCount UNSIGNED INTEGER NOT NULL DEFAULT 0,
	dislikeCount INTEGER NOT NULL DEFAULT 0,
	CHECK(likeCount >= 0),
	CHECK(dislikeCount >= 0),
	FOREIGN KEY(parentCommentID) REFERENCES social_comment(commentID),
	FOREIGN KEY(commenterID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(postID)
);

CREATE TABLE IF NOT EXISTS social_post_like (
	postID INTEGER NOT NULL,
	likerID INTEGER NOT NULL,
	UNIQUE(postID, likerID),
	FOREIGN KEY(likerID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(postID)
);

CREATE TABLE IF NOT EXISTS social_post_dislike (
	postID INTEGER NOT NULL,
	dislikerID INTEGER NOT NULL,
	UNIQUE(postID, dislikerID),
	FOREIGN KEY(dislikerID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(postID)
);

CREATE TABLE IF NOT EXISTS social_comment_like (
	postID INTEGER NOT NULL,
	likerID INTEGER NOT NULL,
	UNIQUE(postID, likerID),
	FOREIGN KEY(likerID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(postID)
);

CREATE TABLE IF NOT EXISTS social_comment_dislike (
	postID INTEGER NOT NULL,
	dislikerID INTEGER NOT NULL,
	UNIQUE(postID, dislikerID),
	FOREIGN KEY(dislikerID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(postID)
);
`

type Post struct {
	postID int64
	poster *util.Account
	postText string
	postTime time.Time
	commentCount util.AtomicUint
	likeCount util.AtomicUint
	dislikeCount util.AtomicUint
}

type Comment struct {
	commentID int64
	commenter *util.Account
	commentText string
	commentTime time.Time
	parentComment *Comment
	childrenCount util.AtomicUint
	likeCount util.AtomicUint
	dislikeCount util.AtomicUint
}

var IDToPost util.SyncMap[int64, *Post]
var IDToComment util.SyncMap[int64, *Comment]

func init() {
	util.SQLInitScript( "social", socialSQLTables )
	IDToPost.Init()
	IDToComment.Init()
}

func TestScript() {
	//TODO: make community system
	op := util.EmailToAccount.GetI("pedro@manse.dev")
	p1 := createPost(op, "# Olá, mundo!")
	c1 := createComment(op, "Olá cara", p1, nil)
	c1_1 := createComment(op, "Olá caras", p1, c1)
	c2 := createComment(op, "Olá gente", p1, nil)
	fmt.Printf("%p=%+v\n\n", p1, p1)
	fmt.Printf("%p=%+v\n\n", c1, c1)
	fmt.Printf("%p=%+v\n\n", c1_1, c1_1)
	fmt.Printf("%p=%+v\n\n", c2, c2)
}

func createComment(creator *util.Account, commentText string, parentPost *Post, parentComment *Comment) (c *Comment) {
	if (parentComment == nil) {
		c = _createSoleComment(creator, commentText, parentPost)
	} else {
		c = _createChildComment(creator, commentText, parentPost, parentComment)
		parentComment.childrenCount.Add(1)
	}
	IDToComment.Set(c.commentID, c)
	_, e := util.SQLDo("service/social.createPost", `
	UPDATE social_post
	SET commentCount = commentCount + 1
	WHERE postID=?;
	`, parentPost.postID)
	if (e != nil) {panic(e)}
	return
}

// posterID INTEGER NOT NULL,
// postText TEXT NOT NULL,
// postTime DATETIME NOT NULL,
func createPost(creator *util.Account, postText string) *Post {
	r, e := util.SQLDo("service/social.createPost", `
	INSERT INTO social_post
	(posterID, postText, postTime)
	VALUES (?, ?, CURRENT_TIMESTAMP)`, creator.ID, postText)
	if (e != nil) {panic(e)}
	PostId, _ := r.LastInsertId()
	p := &Post{
		PostId, creator, postText,
		time.Now(),
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
	}
	IDToPost.Set(PostId, p)
	return p
}

func _createSoleComment(creator *util.Account, commentText string, parentPost *Post) *Comment {
	r, e := util.SQLDo("service/social.createSoleComment", `
	INSERT INTO social_comment
	(commenterID, commentText, commentTime, postID)
	VALUES (?, ?, CURRENT_TIMESTAMP, ?);`, creator.ID, commentText, parentPost.postID)
	if (e != nil) {panic(e)}
	CommentID, _ := r.LastInsertId()
	c := &Comment{
		CommentID,
		creator,
		commentText,
		time.Now(),
		nil,
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
	}
	return c
}

func _createChildComment(creator *util.Account, commentText string, parentPost *Post, parentComment *Comment) *Comment {
	r, e := util.SQLDo("service/social.createChildComment", `
	INSERT INTO social_comment
	(commenterID, commentText, commentTime, postID, parentCommentID)
	VALUES (?, ?, CURRENT_TIMESTAMP, ?, ?);

	UPDATE social_comment
	SET childrenCount = childrenCount + 1
	WHERE commentID=?;
	`, creator.ID, commentText, parentPost.postID, parentComment.commentID, parentComment.commentID)
	if (e != nil) {panic(e)}
	parentPost.commentCount.Add(1)

	CommentID, _ := r.LastInsertId()
	c := &Comment{
		CommentID,
		creator,
		commentText,
		time.Now(),
		parentComment,
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
	}
	return c
}

