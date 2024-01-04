package service

import (
	. "mysrv/util"
	"time"
	"fmt"
	"database/sql"
	"sync/atomic"
	"html/template"
	"github.com/gomarkdown/markdown"

	"io"
	"os"
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
	UNIQUE(postID, accID),
	FOREIGN KEY(accID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(commentID)
);

CREATE TABLE IF NOT EXISTS social_comment_reaction (
	commentID INTEGER NOT NULL,
	accID INTEGER NOT NULL,
	action INTEGER NOT NULL,
	UNIQUE(commentID, accID),
	FOREIGN KEY(commentID) REFERENCES social_comment(commentID),
	FOREIGN KEY(accID) REFERENCES accounts(id)
);
`

func TestScript() {
	fmt.Println(` <!DOCTYPE html> <html> <head> <link href="/files/ex.css" rel="stylesheet" type="text/css"> </head> <body> `)
	//IDToPost.GetI(1).HTMLCard(os.Stdout, IDToAccount.GetI(1))
	IDToPost.GetI(1).HTMLPage(os.Stdout, IDToAccount.GetI(1))
	//IDToPost.GetI(2).HTMLCard(os.Stdout, IDToAccount.GetI(1))
	//IDToPost.GetI(2).HTMLPage(os.Stdout, IDToAccount.GetI(1))
	fmt.Println(`</body> </html>`)
}

func (P *Post) HTMLPage(w io.Writer, viewer *Account) {
	e := template.Must(template.New("Post.Page").Parse(`
<div class="post page" id="post-{{ .post.PostID }}">
	<span class="op">
		<span class="name">{{.post.Poster.Name}}</span>
		<span class="email">{{.post.Poster.Email}}</span>
	</span>
	<div class="content">
		{{ .post.PostHTML }}
	</div>
	{{ $userReaction := .post.Reactions.GetI .acc.ID }}
	<div id="reactions" class="reactions">
		<span class="reactionCount">
			<span class="likes"> {{ .post.LikeCount.Load }} </span>
			<span class="dislikes"> {{ .post.DislikeCount.Load }} </span>
		</span>
		<span class="react">
			{{ range $i, $info := .allreactions }}
				{{ $info.HTML $userReaction $.post.PostID }}
			{{ end }}
		</span>
		<span class="commentCount">
		{{ .post.CommentCount.Load }}
		</span>
		<div class="comments">
			{{ range $i, $comment := .post.Comments }}
				{{ $userReaction := $comment.Reactions.GetI $.acc.ID }}
				{{ $comment.HTML $userReaction $.acc.ID }}
			{{ end }}
		</div>
	</div>
</div>
`)).Execute(w, map[string]any{
	"allreactions":ReactionsInfo,
	"post":*P,
	"acc":viewer,
})
	if (e!=nil) { panic(e) }
}

func (P *Post) HTMLCard(w io.Writer, viewer *Account) {
	e := template.Must(template.New("Post.Card").Parse(`
<div class="post card" id="post-{{ .post.PostID }}">
	<span class="reactionCount">
		<span class="likes"> {{ .post.LikeCount.Load }} </span>
		<span class="dislikes"> {{ .post.DislikeCount.Load }} </span>
	</span>
	{{ .post.PostHTML }}

	{{ $userReaction := .post.Reactions.GetI .acc.ID }}
	<div class="reactions">
		<span class="reactionCount">
			<span class="likes"> {{ .likeCount }} </span>
			<span class="dislikes"> {{ .dislikeCount }} </span>
		</span>
		<span class="react">
			{{ range $i, $info := .allreactions }}
				{{ $info.HTML $userReaction $.post.PostID }}
			{{ end }}
		</span>
		<span class="commentCount">
		{{ .post.CommentCount.Load }}
		</span>
	</div>
</div>
`,
)).Execute(w, map[string]any{
	"allreactions":ReactionsInfo,
	"post":*P,
	"acc":viewer,
})
	if (e!=nil) { panic(e) }
}

func MDToHTML(MD string) (HTML template.HTML) {
	return template.HTML(string(markdown.ToHTML([]byte(MD), nil, nil)))
}

type ReactionType = uint64
type ReactionInfo struct {
	ID ReactionType
	Name string
	MeansHappy bool
	Img string
	Alt string
	AltStyle string
}

func (R ReactionInfo) HTML(selected uint64, postid int64) (template.HTML) {
	if (R.ID == 0) {return ""}
	class:="notSelected"
	if selected == R.ID {
		class="selected"
	}
	return template.HTML(fmt.Sprintf(`
	<button title=%q style=%q class="reaction %s" hx-post="/social/post/react/?reaction=%d&postid=%d">
		<img src=%q alt=%q>
	</button>
	`, R.Name, R.AltStyle, class, R.ID, postid, R.Img, R.Alt) )
}

const (
	NoReaction = iota
	ReactionLike
	ReactionDislike
	ReactionLove
	ReactionHate
	ReactionLaugh
	ReactionLimit
)

var ReactionsInfo = [...]ReactionInfo{
	ReactionLike: ReactionInfo{ ReactionLike, "Like", true,
	"/files/img/reaction_like", "↑", "color: green;",
	},
	ReactionDislike: ReactionInfo{ ReactionDislike, "Dislike", false,
		"/files/img/reaction_dislike", "↓", "color: red;",
	},
	ReactionLove: ReactionInfo{ ReactionLove, "Love", true,
		"/files/img/reaction_love", "<3", "color: pink;",
	},
	ReactionHate: ReactionInfo{ ReactionHate, "Hate", false,
		"/files/img/reaction_hate", "`^´", "color: red;",
	},
	ReactionLaugh: ReactionInfo{ ReactionLaugh, "Laugh", true,
		"/files/img/reaction_laugh", "XD", "color: white;",
	},
}

type Community struct {
	CommunityID int64
	Creator *Account
	Name string
	Description string
	Posts []*Post
	Subscount atomic.Uint64
}

type Post struct {
	PostID int64
	Poster *Account
	Community *Community
	PostText string
	PostHTML template.HTML // markdown -> html!
	PostTime time.Time
	Comments []*Comment
	CommentCount *atomic.Uint64
	LikeCount *atomic.Uint64
	DislikeCount *atomic.Uint64
	Reactions *SyncMap[int64, ReactionType]
}

type Comment struct {
	CommentID int64
	Commenter *Account
	Post *Post
	CommentText string
	CommentHTML template.HTML
	CommentTime time.Time
	ParentComment *Comment
	Children []*Comment
	ChildrenCount *atomic.Uint64
	LikeCount *atomic.Uint64
	DislikeCount *atomic.Uint64
	Reactions *SyncMap[int64, ReactionType]
}

func (C Comment) HTML(reaction uint64, viewer int64) template.HTML {
	return template.HTML(fmt.Sprintf(`
	<div class="comment" id="comment-%d">
	<span class="op">
		<span class="name">%s</span>
	</span>
		<span class="email">%s</span>
	<div class="content">
		%s
	</div>
	%s
	</div>
	`, C.CommentID,
	C.Commenter.Name, C.Commenter.Email,
	C.CommentText, C.HTMLChildren(viewer, 0),
))
}

func (C Comment) HTMLChildren(viewer int64, depth int) string {
	cum := ""
	for _, cmt := range C.Children {
		r := cmt.Reactions.GetI(viewer)
		cum += cmt.ChildHTML(r, viewer, depth)
	}
	return `<div class="children">`+cum+`</div>`
}

func (C Comment) ChildHTML(reaction uint64, viewer int64, depth int) string {
	return fmt.Sprintf(`
	<div class="comment border_%d" id="comment-%d">
	<span class="op">
		<span class="name">%s</span>
		<span class="email">%s</span>
	</span>
	<div class="content">
		%s
	</div>
	%s
	</div>
	`, (depth%4)+1, C.CommentID,
	C.Commenter.Name, C.Commenter.Email,
	C.CommentText, C.HTMLChildren(viewer, depth+1),
)
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

func (C *Comment) React(accID int64, reaction ReactionType) {
	e, _ := SQLDo("service/social.(*Comment).React", `
	INSERT INTO social_comment_reaction
		(commentID, accID, action)
	VALUES
		(?, ?, ?)
	ON CONFLICT
		UPDATE action=?;`,
	C.CommentID, accID, reaction, reaction)
	if (e != nil) {panic(e)}
}

func (P *Post) React(accID int64, reaction ReactionType) {
	e, _ := SQLDo("service/social.(*Post).React", `
	INSERT INTO social_post_reaction
		(postID, accID, action)
	VALUES
		(?, ?, ?)
	ON CONFLICT
		UPDATE action=?;`,
	P.PostID, accID, reaction, reaction)
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
	subber.ID, comm.CommunityID, comm.CommunityID)

	if (e != nil) {panic(e)}
	_loadsubTo(subber, comm)
}

func createComment(creator *Account, commentText string, parentPost *Post, parentComment *Comment) (c *Comment) {
	if (parentComment == nil) {
		c = _createSoleComment(creator, commentText, parentPost)
	} else {
		c = _createChildComment(creator, commentText, parentPost, parentComment)
		parentComment.ChildrenCount.Add(1)
	}
	parentPost.CommentCount.Add(1)
	IDToComment.Set(c.CommentID, c)
	_, e := SQLDo("service/social.createComment", `
	UPDATE social_post
	SET commentCount = commentCount + 1
	WHERE postID=?;
	`, parentPost.PostID)
	//CommentToPost.Set(c, parentPost)
	parentPost.Comments = append(parentPost.Comments, c)
	if (e != nil) {panic(e)}
	return
}

// posterID INTEGER NOT NULL,
// postText TEXT NOT NULL,
// postTime DATETIME NOT NULL,
func createPost(creator *Account, PostText string, comm *Community) *Post {
	r, e := SQLDo("service/social.createPost", `
	INSERT INTO social_post
	(posterID, communityID, postText, postTime)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP);`, creator.ID, comm.CommunityID, PostText)
	if (e != nil) {panic(e)}
	PostId, _ := r.LastInsertId()
	var reactions SyncMap[int64, ReactionType]
	reactions.Init()
	p := &Post{
		PostId, creator, comm,
		PostText,
		MDToHTML(PostText),
		time.Now(),
		[]*Comment{},
		&atomic.Uint64{},
		&atomic.Uint64{},
		&atomic.Uint64{},
		&reactions,
	}
	IDToPost.Set(PostId, p)
	comm.Posts = append(comm.Posts, p)
	return p
}

func _createSoleComment(creator *Account, commentText string, parentPost *Post) *Comment {
	r, e := SQLDo("service/social.createSoleComment", `
	INSERT INTO social_comment
	(commenterID, commentText, commentTime, postID)
	VALUES (?, ?, CURRENT_TIMESTAMP, ?);`, creator.ID, commentText, parentPost.PostID)
	if (e != nil) {panic(e)}
	CommentID, _ := r.LastInsertId()
	var reactions SyncMap[int64, ReactionType]
	reactions.Init()
	c := &Comment{
		CommentID,
		creator,
		parentPost,
		commentText,
		MDToHTML(commentText),
		time.Now(),
		nil,
		[]*Comment{},
		&atomic.Uint64{},
		&atomic.Uint64{},
		&atomic.Uint64{},
		&reactions,
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
	`, creator.ID, commentText, parentPost.PostID, parentComment.CommentID, parentComment.CommentID)
	if (e != nil) {panic(e)}

	var reactions SyncMap[int64, ReactionType]
	reactions.Init()
	CommentID, _ := r.LastInsertId()
	c := &Comment{
		CommentID,
		creator,
		parentPost,
		commentText,
		MDToHTML(commentText),
		time.Now(),
		parentComment,
		[]*Comment{},
		&atomic.Uint64{},
		&atomic.Uint64{},
		&atomic.Uint64{},
		&reactions,
	}
	parentComment.Children = append(parentComment.Children, c)
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
		err = rows.Scan(&commID, &subCount)
		if (err != nil) {return err}
		comm, ok := IDToCommunity.Get(commID)
		if (!ok) {return fmt.Errorf("Can't find community [%d]", commID)}
		comm.Subscount.Store(subCount)
	}
	return
}

func _loadreactions(db *sql.DB) (err error) {
	rows, err := SQLGet("service/social.loadSQL#load reactions for posts", `
	SELECT accID, postID, action FROM social_post_reaction;
	`)
	if (err != nil) {return err}
	defer rows.Close()
	var accID, PostID int64
	var reaction ReactionType

	for rows.Next() {
		err = rows.Scan(&accID, &PostID, &reaction)
		if (err != nil) {return err}
		post, ok := IDToPost.Get(PostID)
		if (!ok) {return fmt.Errorf("Can't find post [%d]", PostID)}
		ok = IDToAccount.Has(accID)
		if (!ok) {return fmt.Errorf("Can't find account [%d]", accID)}
		if (reaction >= ReactionLimit) {
			return fmt.Errorf("Inexistent Reaction [%d]", reaction)
		}
		post.Reactions.Set(accID, reaction)
		if (ReactionsInfo[reaction].MeansHappy) {
			post.LikeCount.Add(1)
		} else {
			post.DislikeCount.Add(1)
		}
	}

	rows, err = SQLGet("service/social.loadSQL#load reactions for comments", `
	SELECT accID, commentID, action FROM social_comment_reaction;
	`)
	if (err != nil) {return err}
	defer rows.Close()
	var commentID int64

	for rows.Next() {
		err = rows.Scan(&accID, &commentID, &reaction)
		if (err != nil) {return err}
		comment, ok := IDToComment.Get(commentID)
		if (!ok) {return fmt.Errorf("Can't find comment [%d]", commentID)}
		ok = IDToAccount.Has(accID)
		if (!ok) {return fmt.Errorf("Can't find account [%d]", accID)}
		if (reaction >= ReactionLimit) {
			return fmt.Errorf("Inexistent Reaction [%d]", reaction)
		}
		comment.Reactions.Set(accID, reaction)
		if (ReactionsInfo[reaction].MeansHappy) {
			comment.LikeCount.Add(1)
		} else {
			comment.DislikeCount.Add(1)
		}
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
		var PostID, communityID, posterID int64
		var PostText string
		var PostTime time.Time
		var commentCount uint64
		err = rows.Scan(
			&PostID, &communityID, &posterID,
			&PostText,
			&PostTime,
			&commentCount,
		)
		if (err != nil){return}
		Poster, ok := IDToAccount.Get(posterID)
		if (!ok) {return fmt.Errorf("Can't find post creator [%d]", posterID)}
		Comm, ok := IDToCommunity.Get(communityID)
		if (!ok) {return fmt.Errorf("Can't find community [%d]", communityID)}
		atomicCommentCount := atomic.Uint64{}
		atomicCommentCount.Store(commentCount)
		reactions := NewSyncMap[int64, ReactionType]()
		p := &Post{
			PostID, Poster, Comm,
			PostText,
			MDToHTML(PostText),
			PostTime,
			[]*Comment{},
			&atomicCommentCount,
			&atomic.Uint64{},
			&atomic.Uint64{},
			&reactions,
		}
		IDToPost.Set(PostID, p)
		Comm.Posts = append(Comm.Posts, p)
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
		var commentID, commenterID, PostID int64
		var parentCommentID *int64
		var commentText string
		var commentTime time.Time
		var childrenCount uint64
		e:=rows.Scan(
			&commentID, &commenterID, &PostID, &parentCommentID,
			&commentText,
			&commentTime,
			&childrenCount,
		)
		if (e != nil) {panic(e)}
		commenter, ok := IDToAccount.Get(commenterID)
		if (!ok) {return fmt.Errorf("Can't find commenter [%d]", commenterID)}
		post, ok := IDToPost.Get(PostID)
		if (!ok) {return fmt.Errorf("Can't find post for comment [%d]", PostID)}
		var reactions SyncMap[int64, ReactionType]
		reactions.Init()
		atomicChildrenCount := atomic.Uint64{}
		atomicChildrenCount.Store(childrenCount)
		c := &Comment{
			commentID, commenter, post,
			commentText,
			MDToHTML(commentText),
			commentTime,
			nil,
			[]*Comment{},
			&atomicChildrenCount,
			&atomic.Uint64{},
			&atomic.Uint64{},
			&reactions,
		}
		if (parentCommentID != nil) {
			setparent = append(setparent, ParentChild{commentID, *parentCommentID})
		} else {
			post.Comments = append(post.Comments, c)
		}
		IDToComment.Set(commentID, c)
	}

	for _, pair := range setparent {
		childID, parentID := pair.Unpack()
		child, ok := IDToComment.Get(childID)
		if (!ok) {return fmt.Errorf("Can't find child comment [%d]", childID)}
		parent, ok := IDToComment.Get(parentID)
		if (!ok) {return fmt.Errorf("Can't find parent comment [%d]", parentID)}
		parent.Children = append(parent.Children, child)
		child.ParentComment = parent
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

