package service

import (
	"mysrv/util"
	"time"
	"fmt"
	"database/sql"
)

/*
DROP TABLE social_community;
DROP TABLE social_sub;
DROP TABLE social_post;
DROP TABLE social_comment;
DROP TABLE social_post_like;
DROP TABLE social_post_dislike;
DROP TABLE social_comment_like;
DROP TABLE social_comment_dislike;
*/

var socialSQLTables = `

CREATE TABLE IF NOT EXISTS social_community (
	communityID INTEGER PRIMARY KEY AUTOINCREMENT,
	creatorID INTEGER NOT NULL,
	name TEXT NOT NULL,
	description TEXT NOT NULL,
	subcount INGETER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS social_sub (
	subberID INTEGER NOT NULL,
	communityID INTEGER NOT NULL,
	UNIQUE(subberID, communityID),
	FOREIGN KEY(subberID) REFERENCES accounts(id),
	FOREIGN KEY(communityID) REFERENCES social_community(communityID)
);

CREATE TABLE IF NOT EXISTS social_post (
	postID INTEGER PRIMARY KEY AUTOINCREMENT,
	communityID INTEGER NOT NULL,
	posterID INTEGER NOT NULL,
	postText TEXT NOT NULL,
	postTime DATETIME NOT NULL,
	commentCount INTEGER NOT NULL DEFAULT 0,
	likeCount INTEGER NOT NULL DEFAULT 0,
	dislikeCount INTEGER NOT NULL DEFAULT 0,
	CHECK(likeCount >= 0),
	CHECK(dislikeCount >= 0),
	FOREIGN KEY(posterID) REFERENCES accounts(id),
	FOREIGN KEY(communityID) REFERENCES social_community(communityID)
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
	commentID INTEGER NOT NULL,
	likerID INTEGER NOT NULL,
	UNIQUE(commentID, likerID),
	FOREIGN KEY(likerID) REFERENCES accounts(id),
	FOREIGN KEY(commentID) REFERENCES social_post(commentID)
);

CREATE TABLE IF NOT EXISTS social_comment_dislike (
	commentID INTEGER NOT NULL,
	dislikerID INTEGER NOT NULL,
	UNIQUE(commentID, dislikerID),
	FOREIGN KEY(dislikerID) REFERENCES accounts(id),
	FOREIGN KEY(commentID) REFERENCES social_post(commentID)
);
`

type Community struct {
	communityID int64
	creator *util.Account
	name string
	description string
	posts []*Post
	subscount util.AtomicUint
}

type Post struct {
	postID int64
	poster *util.Account
	community *Community
	postText string
	postTime time.Time
	comments []*Comment
	commentCount util.AtomicUint
	likeCount util.AtomicUint
	dislikeCount util.AtomicUint
}

type Comment struct {
	commentID int64
	commenter *util.Account
	post *Post
	commentText string
	commentTime time.Time
	parentComment *Comment
	children []*Comment
	childrenCount util.AtomicUint
	likeCount util.AtomicUint
	dislikeCount util.AtomicUint
}

// maps
var (
	IDToPost util.SyncMap[int64, *Post]
	IDToComment util.SyncMap[int64, *Comment]
	IDToCommunity util.SyncMap[int64, *Community]

	// user ID -> sub list
	UIDToSubs util.SyncMap[*util.Account, []*Community]
	// community ID -> sub list
	CIDToSubs util.SyncMap[*Community, []*util.Account]
)

func init() {
	util.SQLInitScript( "social#create tables", socialSQLTables )
	util.SQLInitFunc( "social#load", loadSQL )

	IDToPost.Init()
	IDToComment.Init()
	IDToCommunity.Init()
	UIDToSubs.Init()
	CIDToSubs.Init()
	//CommentToPost.Init()
}

func TestScript() {
	//op := util.EmailToAccount.GetI("test_pedro@manse.dev")
	//cmm := createCommunity(op, "SocAll", "Social community for all")
	//p1 := createPost(op, "# Olá, mundo!", cmm)
	//c1 := createComment(op, "Olá cara", p1, nil)
	//_ = createComment(op, "Olá caras", p1, c1)
	//_ = createComment(op, "Olá gente", p1, nil)
	fmt.Println(IDToPost)
	fmt.Println(IDToComment)
	fmt.Println(IDToCommunity)
	fmt.Println(UIDToSubs)
	fmt.Println(CIDToSubs)
}

func createCommunity(creator *util.Account, name string, description string) (c *Community) {
	r, e := util.SQLDo("service/social.createCommunity", `
	INSERT INTO social_community (creatorID, name, description)
	VALUES (?, ?, ?); `, creator.ID, name, description)

	if (e != nil) {panic(e)}
	commID, _ := r.LastInsertId()
	c = &Community{
		commID, creator,
		name, description,
		[]*Post{},
		util.NewAtomicUint(0),
	}

	subTo(creator, c)
	IDToCommunity.Set(commID, c)

	return
}

func _loadsubTo(subber *util.Account, comm *Community) {
	csublist, ok := CIDToSubs.Get(comm)
	if (!ok) {
		csublist = []*util.Account{subber}
	} else {
		csublist = append(csublist, subber)
	}
	CIDToSubs.Set(comm, csublist)

	usublist, ok := UIDToSubs.Get(subber)
	if (!ok) {
		usublist = []*Community{comm}
	} else {
		usublist = append(usublist, comm)
	}
	UIDToSubs.Set(subber, usublist)
}

func subTo(subber *util.Account, comm *Community) {
	_, e := util.SQLDo("service/social.createCommunity#setCreatorAsSub", `
	INSERT INTO social_sub (subberID, communityID)
	VALUES (?, ?);

	UPDATE social_community SET subcount=subcount+1 WHERE communityID=?;`,
	subber.ID, comm.communityID, comm.communityID)

	if (e != nil) {panic(e)}
	_loadsubTo(subber, comm)
}

func createComment(creator *util.Account, commentText string, parentPost *Post, parentComment *Comment) (c *Comment) {
	if (parentComment == nil) {
		c = _createSoleComment(creator, commentText, parentPost)
	} else {
		c = _createChildComment(creator, commentText, parentPost, parentComment)
		parentComment.childrenCount.Add(1)
	}
	IDToComment.Set(c.commentID, c)
	_, e := util.SQLDo("service/social.createComment", `
	UPDATE social_post
	SET commentCount = commentCount + 1
	WHERE postID=?;
	`, parentPost.postID)
	//CommentToPost.Set(c, parentPost)
	parentPost.comments = append(parentPost.comments, c)
	if (e != nil) {panic(e)}
	return
}

// posterID INTEGER NOT NULL,
// postText TEXT NOT NULL,
// postTime DATETIME NOT NULL,
func createPost(creator *util.Account, postText string, comm *Community) *Post {
	r, e := util.SQLDo("service/social.createPost", `
	INSERT INTO social_post
	(posterID, communityID, postText, postTime)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, creator.ID, comm.communityID, postText)
	if (e != nil) {panic(e)}
	PostId, _ := r.LastInsertId()
	p := &Post{
		PostId, creator, comm,
		postText,
		time.Now(),
		[]*Comment{},
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
	}
	IDToPost.Set(PostId, p)
	comm.posts = append(comm.posts, p)
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
		parentPost,
		commentText,
		time.Now(),
		nil,
		[]*Comment{},
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
		parentPost,
		commentText,
		time.Now(),
		parentComment,
		[]*Comment{},
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
		util.NewAtomicUint(0),
	}
	parentComment.children = append(parentComment.children, c)
	return c
}

func _loadsubs(db *sql.DB) (err error) {
	return
}
func _countreactions(db *sql.DB) (err error) {
	return
}
func _loadcomm(db *sql.DB) (err error) {
	rows, err := db.Query(`
SELECT communityID, creatorID, name, description,
	(SELECT COUNT(subberID)
	FROM social_sub
	WHERE communityID=social_community.communityID)
FROM social_community`)
	if (err != nil) {return}
	for rows.Next() {
		var communityID, creatorID int64
		var name, description string
		var subc uint64
		err = rows.Scan(&communityID, &creatorID, &name, &description, &subc)
		if (err != nil) {return}

		acc, ok := util.IDToAccount.Get(creatorID)
		if (!ok) {return fmt.Errorf("Can't find community creator [%d]\n", creatorID)}
		c := &Community{
			communityID, acc,
			name, description,
			[]*Post{},
			util.NewAtomicUint(subc),
		}

		_loadsubTo(acc, c)
		IDToCommunity.Set(communityID, c)
	}
	return
}

func _loadposts(db *sql.DB) (err error) {
	rows, err := db.Query(`
SELECT
	postID, communityID, posterID, postText, postTime,
	commentCount, likeCount, dislikeCount,
	(
		SELECT COUNT(likerID)
		FROM social_post_like
		WHERE postID=social_post.postID
	),
	(
		SELECT COUNT(dislikerID)
		FROM social_post_dislike
		WHERE postID=social_post.postID
	)
FROM
	social_post;
`)
	if (err != nil){return}
	for rows.Next() {
		var postID, communityID, posterID int64
		var postText string
		var postTime time.Time
		var commentCount, likeCount, dislikeCount uint64
		var sqllikeCount, sqldislikeCount uint64
		err = rows.Scan(
			&postID, &communityID, &posterID,
			&postText,
			&postTime,
			&commentCount, &likeCount, &dislikeCount,
			&sqllikeCount, &sqldislikeCount,

		)
		if (likeCount != sqllikeCount) {
			return fmt.Errorf("like counter for post is wrong [%d]\n", postID)
		}
		if (dislikeCount != sqldislikeCount) {
			return fmt.Errorf("dislike counter for post is wrong [%d]\n", postID)
		}
		if (err != nil){return}
		poster, ok := util.IDToAccount.Get(posterID)
		if (!ok) {return fmt.Errorf("Can't find post creator [%d]\n", posterID)}
		comm, ok := IDToCommunity.Get(communityID)
		if (!ok) {return fmt.Errorf("Can't find community [%d]\n", communityID)}
		p := &Post{
			postID, poster, comm,
			postText, postTime,
			[]*Comment{},
			util.NewAtomicUint(commentCount),
			util.NewAtomicUint(likeCount),
			util.NewAtomicUint(dislikeCount),
		}
		IDToPost.Set(postID, p)
		comm.posts = append(comm.posts, p)
	}
	return
}

type ParentChild = util.Tuple[int64, int64]
func _loadcomments(db *sql.DB) (err error) {
	rows, err := db.Query(`
SELECT
	commentID, commenterID, postID, parentCommentID,
	commentText,
	commentTime,
	childrenCount, likeCount, dislikeCount,
	(
		SELECT COUNT(likerID)
		FROM social_comment_like
		WHERE commentID=social_comment.commentID
	),
	(
		SELECT COUNT(dislikerID)
		FROM social_comment_dislike
		WHERE commentID=social_comment.commentID
	)
FROM
	social_comment;
`)
	if (err != nil){return}
	// cmt[prntcmt] = prnt
	var setparent = []ParentChild{}

	for rows.Next() {
		var commentID, commenterID, postID, parentCommentID int64
		var commentText string
		var commentTime time.Time
		var childrenCount, likeCount, dislikeCount uint64
		var sqllikeCount, sqldislikeCount uint64
		rows.Scan(
			&commentID, &commenterID, &postID, &parentCommentID,
			&commentText,
			&commentTime,
			&childrenCount, &likeCount, &dislikeCount,
			&sqllikeCount, &sqldislikeCount,
		)
		if (likeCount != sqllikeCount) {
			return fmt.Errorf("like counter for comment is wrong [%d]\n", commentID)
		}
		if (dislikeCount != sqldislikeCount) {
			return fmt.Errorf("dislike counter for comment is wrong [%d]\n", commentID)
		}
		commenter, ok := util.IDToAccount.Get(commenterID)
		if (!ok) {return fmt.Errorf("Can't find commenter [%d]\n", commenterID)}
		post, ok := IDToPost.Get(postID)
		if (!ok) {return fmt.Errorf("Can't find post for comment [%d]\n", postID)}
		if (parentCommentID != 0) {
			setparent = append(setparent, ParentChild{commentID, parentCommentID})
		}
		c := &Comment{
			commentID, commenter, post,
			commentText,
			commentTime,
			nil,
			[]*Comment{},
			util.NewAtomicUint(childrenCount),
			util.NewAtomicUint(likeCount),
			util.NewAtomicUint(dislikeCount),
		}
		IDToComment.Set(commentID, c)
		post.comments = append(post.comments, c)
	}

	for _, pair := range setparent {
		childID, parentID := pair.Unpack()
		child, ok := IDToComment.Get(childID)
		if (!ok) {return fmt.Errorf("Can't find child comment [%d]\n", childID)}
		parent, ok := IDToComment.Get(parentID)
		if (!ok) {return fmt.Errorf("Can't find parent comment [%d]\n", parentID)}
		parent.children = append(parent.children, child)
		child.parentComment = parent
	}
	return
}

func loadSQL(db *sql.DB) (err error) {
	err = _loadcomm(db)
	if (err != nil) {return}
	err = _loadposts(db)
	if (err != nil) {return}
	err = _loadcomments(db)
	if (err != nil) {return}
	return
}

