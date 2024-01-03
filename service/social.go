package service

import (
	. "mysrv/util"
	"time"
	"fmt"
	"database/sql"
	"sync/atomic"
)

/*
DROP TABLE social_community;
DROP TABLE social_sub;
DROP TABLE social_post;
DROP TABLE social_comment;
DROP TABLE social_post_reaction;
DROP TABLE social_comment_reaction;
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
	FOREIGN KEY(parentCommentID) REFERENCES social_comment(commentID),
	FOREIGN KEY(commenterID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(postID)
);

CREATE TABLE IF NOT EXISTS social_post_reaction (
	postID INTEGER NOT NULL,
	accID INTEGER NOT NULL,
	action INTEGER NOT NULL,
	FOREIGN KEY(accID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(commentID)
);

CREATE TABLE IF NOT EXISTS social_comment_reaction (
	commentID INTEGER NOT NULL,
	accID INTEGER NOT NULL,
	action INTEGER NOT NULL,
	FOREIGN KEY(commentID) REFERENCES social_comment(commentID),
	FOREIGN KEY(accID) REFERENCES accounts(id)
);
`

type ReactionType = uint64
type ReactionInfo struct {
	name string
	meansHappy bool
	img string
}

const (
	ReactionLike = iota
	ReactionDislike
	ReactionLove
	ReactionHate
	ReactionLaugh
	ReactionLimit
)

var ReactionInfos = [...]ReactionInfo{
	ReactionLike: ReactionInfo{"Like", true, "/files/img/reaction_like"},
	ReactionDislike: ReactionInfo{"Dislike", false, "/files/img/reaction_dislike"},
	ReactionLove: ReactionInfo{"Love", true, "/files/img/reaction_love"},
	ReactionHate: ReactionInfo{"Hate", false, "/files/img/reaction_hate"},
	ReactionLaugh: ReactionInfo{"Laugh", true, "/files/img/reaction_laugh"},
}

type Community struct {
	communityID int64
	creator *Account
	name string
	description string
	posts []*Post
	subscount atomic.Uint64
}

type Post struct {
	postID int64
	poster *Account
	community *Community
	postText string
	postTime time.Time
	comments []*Comment
	commentCount atomic.Uint64
	likeCount atomic.Uint64
	dislikeCount atomic.Uint64
	reactions SyncMap[int64, ReactionType]
}

type Comment struct {
	commentID int64
	commenter *Account
	post *Post
	commentText string
	commentTime time.Time
	parentComment *Comment
	children []*Comment
	childrenCount atomic.Uint64
	likeCount atomic.Uint64
	dislikeCount atomic.Uint64
	reactions SyncMap[int64, ReactionType]
}

// maps
var (
	IDToPost SyncMap[int64, *Post]
	IDToComment SyncMap[int64, *Comment]
	IDToCommunity SyncMap[int64, *Community]

	// user ID -> sub list
	UIDToSubs SyncMap[*Account, []*Community]
	// community ID -> sub list
	CIDToSubs SyncMap[*Community, []*Account]
)

func init() {
	SQLInitScript( "social#create tables", socialSQLTables )
	SQLInitFunc( "social#load", loadSQL )

	IDToPost.Init()
	IDToComment.Init()
	IDToCommunity.Init()
	UIDToSubs.Init()
	CIDToSubs.Init()
	//CommentToPost.Init()
}

func ddprt[A any](a *A) {
	fmt.Printf("\n[%p] %#+v\n", a, *a)
}

func dprt[A any](a *A) {
	fmt.Printf("\n[%p] %T %+v\n", a, *a, *a)
}

func TestScript() {
	dprt(IDToPost.GetI(1))
	fmt.Println(IDToPost)
	fmt.Println(IDToComment)
	fmt.Println(UIDToSubs)
	fmt.Println(CIDToSubs)
}

func (P *Post) React(accID int64, reaction ReactionType) {
	e, _ := SQLDo("service/social.(*Post).React", `
	INSERT INTO social_post_reaction
		(postID, accID, action)
	VALUES
		(?, ?, ?)
	ON CONFLICT
		UPDATE action=?;`,
	P.postID, accID, reaction, reaction)
	if (e != nil) {panic(e)}
}

func createCommunity(creator *Account, name string, description string) (c *Community) {
	r, e := SQLDo("service/social.createCommunity", `
	INSERT INTO social_community (creatorID, name, description)
	VALUES (?, ?, ?); `, creator.ID, name, description)

	if (e != nil) {panic(e)}
	commID, _ := r.LastInsertId()
	c = &Community{
		commID, creator,
		name, description,
		[]*Post{},
		atomic.Uint64{},
	}

	subTo(creator, c)
	IDToCommunity.Set(commID, c)

	return
}

func _loadsubTo(subber *Account, comm *Community) {
	csublist, ok := CIDToSubs.Get(comm)
	if (!ok) {
		csublist = []*Account{subber}
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

func subTo(subber *Account, comm *Community) {
	_, e := SQLDo("service/social.createCommunity#setCreatorAsSub", `
	INSERT INTO social_sub (subberID, communityID)
	VALUES (?, ?);

	UPDATE social_community SET subcount=subcount+1 WHERE communityID=?;`,
	subber.ID, comm.communityID, comm.communityID)

	if (e != nil) {panic(e)}
	_loadsubTo(subber, comm)
}

func createComment(creator *Account, commentText string, parentPost *Post, parentComment *Comment) (c *Comment) {
	if (parentComment == nil) {
		c = _createSoleComment(creator, commentText, parentPost)
	} else {
		c = _createChildComment(creator, commentText, parentPost, parentComment)
		parentComment.childrenCount.Add(1)
	}
	IDToComment.Set(c.commentID, c)
	_, e := SQLDo("service/social.createComment", `
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
func createPost(creator *Account, postText string, comm *Community) *Post {
	r, e := SQLDo("service/social.createPost", `
	INSERT INTO social_post
	(posterID, communityID, postText, postTime)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP);`, creator.ID, comm.communityID, postText)
	if (e != nil) {panic(e)}
	PostId, _ := r.LastInsertId()
	var reactions SyncMap[int64, ReactionType]
	reactions.Init()
	p := &Post{
		PostId, creator, comm,
		postText,
		time.Now(),
		[]*Comment{},
		atomic.Uint64{},
		atomic.Uint64{},
		atomic.Uint64{},
		reactions,
	}
	IDToPost.Set(PostId, p)
	comm.posts = append(comm.posts, p)
	return p
}

func _createSoleComment(creator *Account, commentText string, parentPost *Post) *Comment {
	r, e := SQLDo("service/social.createSoleComment", `
	INSERT INTO social_comment
	(commenterID, commentText, commentTime, postID)
	VALUES (?, ?, CURRENT_TIMESTAMP, ?);`, creator.ID, commentText, parentPost.postID)
	if (e != nil) {panic(e)}
	CommentID, _ := r.LastInsertId()
	var reactions SyncMap[int64, ReactionType]
	reactions.Init()
	c := &Comment{
		CommentID,
		creator,
		parentPost,
		commentText,
		time.Now(),
		nil,
		[]*Comment{},
		atomic.Uint64{},
		atomic.Uint64{},
		atomic.Uint64{},
		reactions,
	}
	return c
}

func _createChildComment(creator *Account, commentText string, parentPost *Post, parentComment *Comment) *Comment {
	r, e := SQLDo("service/social.createChildComment", `
	INSERT INTO social_comment
	(commenterID, commentText, commentTime, postID, parentCommentID)
	VALUES (?, ?, CURRENT_TIMESTAMP, ?, ?);

	UPDATE social_comment
	SET childrenCount = childrenCount + 1
	WHERE commentID=?;
	`, creator.ID, commentText, parentPost.postID, parentComment.commentID, parentComment.commentID)
	if (e != nil) {panic(e)}
	parentPost.commentCount.Add(1)

	var reactions SyncMap[int64, ReactionType]
	reactions.Init()
	CommentID, _ := r.LastInsertId()
	c := &Comment{
		CommentID,
		creator,
		parentPost,
		commentText,
		time.Now(),
		parentComment,
		[]*Comment{},
		atomic.Uint64{},
		atomic.Uint64{},
		atomic.Uint64{},
		reactions,
	}
	parentComment.children = append(parentComment.children, c)
	return c
}

func _loadsubs(db *sql.DB) (err error) {
	rows, err := SQLGet("service/social.loadSQL#load subs", `
	SELECT communityID, COUNT() FROM social_sub GROUP BY communityID;
	`)
	if (err != nil) {return err}
	defer rows.Close()
	for rows.Next() {
		var commID int64
		var subCount uint64
		rows.Scan(&commID, &subCount)
		comm, ok := IDToCommunity.Get(commID)
		if (!ok) {return fmt.Errorf("Can't find community [%d]", commID)}
		comm.subscount.Store(subCount)
	}
	return
}

func _loadreactions(db *sql.DB) (err error) {
	rows, err := SQLGet("service/social.loadSQL#load reactions", `
	SELECT accID, postID, action FROM social_post_reaction;
	`)
	if (err != nil) {return err}
	defer rows.Close()
	var accID, postID int64
	var reaction ReactionType

	for rows.Next() {
		err = rows.Scan(&accID, &postID, &reaction)
		if (err != nil) {return err}
		post, ok := IDToPost.Get(postID)
		if (!ok) {return fmt.Errorf("Can't find post [%d]", postID)}
		ok = IDToAccount.Has(accID)
		if (!ok) {return fmt.Errorf("Can't find account [%d]", accID)}
		if (reaction >= ReactionLimit) {
			return fmt.Errorf("Inexistent Reaction [%d]", reaction)
		}
		post.reactions.Set(accID, reaction)
	}
	return
}

func _loadcomm(db *sql.DB) (err error) {
	rows, err := SQLGet("service/social.loadSQL#load communities", `
SELECT communityID, creatorID, name, description,
	(SELECT COUNT(subberID)
	FROM social_sub
	WHERE communityID=social_community.communityID)
FROM social_community`)
	if (err != nil) {return}
	defer rows.Close()
	for rows.Next() {
		var communityID, creatorID int64
		var name, description string
		var subc uint64
		err = rows.Scan(&communityID, &creatorID, &name, &description, &subc)
		if (err != nil) {return}

		acc, ok := IDToAccount.Get(creatorID)
		if (!ok) {return fmt.Errorf("Can't find community creator [%d]", creatorID)}
		atomicSubC := atomic.Uint64{}
		atomicSubC.Store(subc)
		c := &Community{
			communityID, acc,
			name, description,
			[]*Post{},
			atomicSubC,
		}

		_loadsubTo(acc, c)
		IDToCommunity.Set(communityID, c)
	}
	return
}

func _loadposts(db *sql.DB) (err error) {
	rows, err := SQLGet("service/social.loadSQL#load posts", `
SELECT
	postID, communityID, posterID, postText, postTime, commentCount
FROM
	social_post;
`)
	if (err != nil){return}
	defer rows.Close()
	for rows.Next() {
		var postID, communityID, posterID int64
		var postText string
		var postTime time.Time
		var commentCount uint64
		err = rows.Scan(
			&postID, &communityID, &posterID,
			&postText,
			&postTime,
			&commentCount,
		)
		if (err != nil){return}
		poster, ok := IDToAccount.Get(posterID)
		if (!ok) {return fmt.Errorf("Can't find post creator [%d]", posterID)}
		comm, ok := IDToCommunity.Get(communityID)
		if (!ok) {return fmt.Errorf("Can't find community [%d]", communityID)}
		atomicCommentCount := atomic.Uint64{}
		atomicCommentCount.Store(commentCount)
		p := &Post{
			postID, poster, comm,
			postText, postTime,
			[]*Comment{},
			atomicCommentCount,
			atomic.Uint64{},
			atomic.Uint64{},
			NewSyncMap[int64, ReactionType](),
		}
		IDToPost.Set(postID, p)
		comm.posts = append(comm.posts, p)
	}
	return
}

type ParentChild = Tuple[int64, int64]
func _loadcomments(db *sql.DB) (err error) {
	rows, err := SQLGet("service/social.loadSQL#load comments", `
SELECT
	commentID, commenterID, postID, parentCommentID,
	commentText, commentTime, childrenCount
FROM
	social_comment;
`)
	if (err != nil){return}
	defer rows.Close()
	var setparent = []ParentChild{}

	for rows.Next() {
		var commentID, commenterID, postID, parentCommentID int64
		var commentText string
		var commentTime time.Time
		var childrenCount uint64
		rows.Scan(
			&commentID, &commenterID, &postID, &parentCommentID,
			&commentText,
			&commentTime,
			&childrenCount,
		)
		commenter, ok := IDToAccount.Get(commenterID)
		if (!ok) {return fmt.Errorf("Can't find commenter [%d]", commenterID)}
		post, ok := IDToPost.Get(postID)
		if (!ok) {return fmt.Errorf("Can't find post for comment [%d]", postID)}
		if (parentCommentID != 0) {
			setparent = append(setparent, ParentChild{commentID, parentCommentID})
		}
		var reactions SyncMap[int64, ReactionType]
		reactions.Init()
		atomicChildrenCount := atomic.Uint64{}
		atomicChildrenCount.Store(childrenCount)
		c := &Comment{
			commentID, commenter, post,
			commentText,
			commentTime,
			nil,
			[]*Comment{},
			atomicChildrenCount,
			atomic.Uint64{},
			atomic.Uint64{},
			reactions,
		}
		IDToComment.Set(commentID, c)
		post.comments = append(post.comments, c)
	}

	for _, pair := range setparent {
		childID, parentID := pair.Unpack()
		child, ok := IDToComment.Get(childID)
		if (!ok) {return fmt.Errorf("Can't find child comment [%d]", childID)}
		parent, ok := IDToComment.Get(parentID)
		if (!ok) {return fmt.Errorf("Can't find parent comment [%d]", parentID)}
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
	err = _loadsubs(db)
	if (err != nil) {return}
	err = _loadreactions(db)
	if (err != nil) {return}
	return
}

