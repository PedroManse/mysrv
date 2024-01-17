package service

import (
	. "mysrv/util"
	verfit "mysrv/verfitting"
	"time"
	"fmt"
	"database/sql"
	"html/template"
	"github.com/gomarkdown/markdown"
	"strconv"
	"sort"
	"net/http"
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
	CHECK( action > 0),
	UNIQUE(postID, accID),
	FOREIGN KEY(accID) REFERENCES accounts(id),
	FOREIGN KEY(postID) REFERENCES social_post(commentID)
);

CREATE TABLE IF NOT EXISTS social_comment_reaction (
	commentID INTEGER NOT NULL,
	accID INTEGER NOT NULL,
	action INTEGER NOT NULL,
	CHECK( action > 0),
	UNIQUE(commentID, accID),
	FOREIGN KEY(accID) REFERENCES accounts(id),
	FOREIGN KEY(commentID) REFERENCES social_comment(commentID)
);
`

var AllEndpoint = LogicPage(
	"html/social/cards.gohtml",
	map[string]any{
		"allreactions":ReactionsInfo,
		"allsorts":SortNames,
	},
	[]GOTMPlugin{ GOTM_account, GOTM_mustacc, GOTM_urlInfo},
	postsEndpoint,
)
var SubbedEndpoint ContentServer = nil
var SavedEndpoint ContentServer = nil

var PostPageEndpoint = LogicPage(
	"html/social/post.gohtml",
	map[string]any{
		"allreactions":ReactionsInfo,
		"allsorts":SortNames,
	},
	[]GOTMPlugin{ GOTM_account, GOTM_mustacc },
	postEndpoint,
)

var CreatePostPageEndpoint = LogicPage(
	"html/social/createpost.gohtml", nil,
	[]GOTMPlugin{ GOTM_account, GOTM_mustacc },
	createpostEndpoint,
)

var ReactToPostEndpoint = LogicPage(
	"html/social/components/post-reactions.gohtml",
	map[string]any{
		"allreactions":ReactionsInfo,
		"allsorts":SortNames,
	},
	[]GOTMPlugin{ GOTM_account, GOTM_mustacc },
	reactToPostEndpoint,
)

var ReactToCommentEndpoint = LogicPage(
	"html/social/components/comment-reactions.gohtml",
	map[string]any{
		"allreactions":ReactionsInfo,
		"allsorts":SortNames,
	},
	[]GOTMPlugin{ GOTM_account, GOTM_mustacc },
	reactToCommentEndpoint,
)

var CreateCommentEndpoint = TemplatedLogicedPluggedPage(
	"html/social/comment-error.gohtml", nil,
	[]GOTMPlugin{GOTM_account},
	createCommentEndpoint,
)

var CompReplyFormEndpoint = TemplatedPluggedPage(
	"html/social/components/replyToCommentForm.gohtml", nil,
	[]GOTMPlugin{GOTM_urlInfo},
)

var CompReplyButtonEndpoint = TemplatedPluggedPage(
	"html/social/components/replyToCommentButton.gohtml", nil,
	[]GOTMPlugin{GOTM_urlInfo},
)

type SocialQuery struct {
	AccID int64
	CommunityID int64
	PostID int64
	PostID_str string
	CommentID int64
	Reaction ReactionType
	PostCount int64
	UseSort SortMethod
	UseSort_str string
}

func prelude(w HttpWriter, r HttpReq, info map[string]any) (acc *Account, query SocialQuery, ok bool) {
	r.ParseForm()
	var accinf = info["acc"].(map[string]any)
	var accid int64
	//TODO: would not need to do this if plugin's terminator flag was implemented
	if (accinf["ok"].(bool)) {
		accid = accinf["id"].(int64)
	} else {
		return
	}

	query.AccID = accid
	acc, ok = IDToAccount.Get(accid)
	if (!ok) {return}
	query.CommunityID, _  = strconv.ParseInt(r.FormValue("communityid"), 10, 64)
	query.PostID, _       = strconv.ParseInt(r.FormValue("postid"), 10, 64)
	query.PostID_str      = r.FormValue("postid")
	query.CommentID, _    = strconv.ParseInt(r.FormValue("commentid"), 10, 64)
	query.Reaction, _     = strconv.ParseUint(r.FormValue("reaction"), 10, 64)
	query.PostCount, _    = strconv.ParseInt(r.FormValue("postcount"), 10, 64)
	query.UseSort         = SortNameToID.GetI(r.FormValue("sortmethod"))
	query.UseSort_str     = r.FormValue("sortmethod")
	if (query.PostCount == 0) {
		query.PostCount = 10
	}
	return
}

func createCommentEndpoint( w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any) {
	acc, query, ok := prelude(w, r, info)
	einfo := Tuple[string, string]{"", `/social/posts?postid=`+query.PostID_str}
	if (!ok) {
		einfo.Left="Invalid Account"
		return true, einfo
	}
	cmnt := IDToComment.GetI(query.CommentID)
	post, ok := IDToPost.Get(query.PostID)
	if (!ok) {
		einfo.Left="Invalid PostID"
		einfo.Right="/social/all"
		return true, einfo
	}

	commentText := r.FormValue("commentText")

	//TODO: better tests on only spaces etc
	if (commentText == "") {
		einfo.Left="Comment with no body"
		return true, einfo
	}

	createComment(acc, commentText, post, cmnt)
	http.Redirect(w, r, fmt.Sprintf("/social/posts?postid=%d", query.PostID), http.StatusFound)
	return false, nil
}

func reactToCommentEndpoint( w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any) {
	_, query, ok := prelude(w, r, info)
	if (!ok) {return true, map[string]any{"error":"Invalid Account"}}
	comment, ok := IDToComment.Get(query.CommentID)
	if (!ok) {return true, map[string]any{"error":"Invalid PostID"}}
	if (r.Method == "POST") {
		comment.React( query.AccID, query.Reaction )
	} else {
		//TODO actually delete reaction
		comment.React( query.AccID, 0 )
	}
	return true, map[string]any{
		"comment":comment,
		"query":query,
	}
}

func reactToPostEndpoint( w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any) {
	_, query, ok := prelude(w, r, info)
	if (!ok) {return true, map[string]any{"error":"Invalid Account"}}
	post, ok := IDToPost.Get(query.PostID)
	if (!ok) {return true, map[string]any{"error":"Invalid PostID"}}
	if (r.Method == "POST") {
		post.React( query.AccID, query.Reaction )
	} else {
		//TODO actually delete reaction
		post.React( query.AccID, 0 )
	}
	return true, map[string]any{
		"post":post,
		"query":query,
	}
}

func createpostEndpoint(w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any) {
	acc, query, ok := prelude(w, r, info)
	if (!ok) {return true, map[string]any{"error":"Invalid Account"}}
	if (r.Method == "GET") {
		return true, map[string]any{}
	} else if (r.Method == "POST") {
		cid := query.CommunityID
		txt := r.FormValue("postText")
		comm, ok := IDToCommunity.Get(cid)
		if (!ok) {return true, map[string]any{"error":"Invalid Community"}}

		fmt.Println(cid)
		fmt.Println(txt)
		pid := createPost(acc, txt, comm).PostID
		http.Redirect(w, r, fmt.Sprintf("/social/posts?postid=%d", pid), http.StatusFound)
		return false, nil
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return false, nil
	}
}

func postEndpoint(w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any) {
	_, query, ok := prelude(w, r, info)
	if (!ok) {return true, map[string]any{"error":"Invalid Account"}}
	pid := query.PostID
	P, ok := IDToPost.Get(pid)
	if (!ok) {return true, map[string]any{"error":"Invalid PostId"}}
	return true, map[string]any{ "post":P  }
}

func postsEndpoint(w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any) {
	_, query, ok := prelude(w, r, info)
	if (!ok) {return}

	torender := []*Post{}
	var i int64 = 0
	// pre-sort by time; gather only PostCount
	for post := range IDToPost.IterValues() {
		torender = append(torender, post)
		i++
	}
	sortedPosts := SortPosts(query.UseSort, torender)
	sortedPosts = sortedPosts[:Min(int(query.PostCount), len(sortedPosts))]

	return true, map[string]any{
		"query": query,
		"posts": sortedPosts,
	}
}

func MDToHTML(MD string) (HTML template.HTML) {
	return template.HTML(string(markdown.ToHTML([]byte((MD)), nil, nil)))
}

type SortMethod = int64
const (
	_ = iota
	SortNewest
	SortOldest
	SortComments
	SortLikes
	SortDislikes
	SortDivided // LikeCout ~= DislikeCount
	SortUnited // LikeCount <<>> DislikeCount
	SortUnitedLove // LikeCount >> DislikeCount
	SortUnitedHate // LikeCount << DislikeCount
)

var SortNameToID = ISyncMap(map[string]SortMethod {
	"Newest":         SortNewest,
	"Oldest":         SortOldest,
	"MostComments":   SortComments,
	"MostLikes":      SortLikes,
	"MostDislikes":   SortDislikes,
	"MostDivided":    SortDivided,
	"MostUnited":     SortUnited,
	"MostUnitedLove": SortUnitedLove,
	"MostUnitedHate": SortUnitedHate,
})

var SortNames = []string {
	"Newest",
	"Oldest",
	"MostComments",
	"MostLikes",
	"MostDislikes",
	"MostDivided",
	"MostUnited",
	"MostUnitedLove",
	"MostUnitedHate",
}

func SortPosts(UseSort SortMethod, PostsToSort []*Post) []*Post {
	sp := SortedPosts{PostsToSort, UseSort}
	sort.Sort(sp)
	return PostsToSort
}

type SortedPosts struct {
	PostsToRender []*Post
	UseSort SortMethod
}

func (S SortedPosts) Len() (int) { return len(S.PostsToRender) }
func (S SortedPosts) Swap(i, j int) {
	S.PostsToRender[i], S.PostsToRender[j] = S.PostsToRender[j], S.PostsToRender[i]
}
func (S SortedPosts) Less(i, j int) bool {
	PostA := S.PostsToRender[i]
	PostB := S.PostsToRender[j]
	if (PostB == nil) { return false }
	if (PostA == nil) { return true }

	//TODO get united/diveded ratio
	switch (S.UseSort) {
	default:
		fallthrough
	case SortNewest:
		return PostA.PostTime.After(PostB.PostTime)
	case SortOldest:
		return PostA.PostTime.Before(PostB.PostTime)
	case SortComments:
		return PostA.CommentCount.Load() > PostB.CommentCount.Load()
	case SortLikes:
		return PostA.LikeCount.Load() > PostB.LikeCount.Load()
	case SortDislikes:
		return PostA.DislikeCount.Load() > PostB.DislikeCount.Load()
	case SortDivided:
		return PostA.DislikeCount.Load() > PostB.DislikeCount.Load()
	case SortUnited:
		return PostA.DislikeCount.Load() > PostB.DislikeCount.Load()
	case SortUnitedLove:
		return PostA.DislikeCount.Load() > PostB.DislikeCount.Load()
	case SortUnitedHate:
		return PostA.DislikeCount.Load() > PostB.DislikeCount.Load()
	}
}

func DebugSocial() {
}

type ReactionType = uint64
type ReactionInfo struct {
	ID ReactionType
	Name string
	MeansHappy bool
	Img string
	Alt string
	AltStyle template.CSS
}

func HTMLCommentReactions(selected uint64, commentid int64) (h template.HTML) {
	hm := ""
	for _, rct := range ReactionsInfo {
		hm+=rct.HTMLstrComment(selected, commentid)
	}
	return template.HTML(hm)
}

func (R ReactionInfo) HTML(selected uint64, postid int64) (template.HTML) {
	return template.HTML(R.HTMLstr(selected, postid))
}

func (R ReactionInfo) HTMLComment(selected uint64, commentid int64) (template.HTML) {
	return template.HTML(R.HTMLstrComment(selected, commentid))
}

var ReactionHTML = InlineUnsafeComponent(`
	<button
		title="{{ .R.Name }}" style="{{ .R.AltStyle }}" class="reaction {{ .class }}"
		{{ .action }}="/social/posts/react?reaction={{ .R.ID }}&postid={{ .postid }}"
		hx-target="#post-{{ .postid }} > span.reactions"
		hx-swap="outerHTML"
	>
		<img src="{{.R.Img}}" alt="{{.R.Alt}}">
	</button>
`)

func (R ReactionInfo) HTMLstr(selected uint64, postid int64) (string) {
	if (R.ID == 0) {return ""}

	class:="notSelected"
	action:="hx-post"
	if selected == R.ID {
		class="selected"
		action="hx-delete"
	}
	return ReactionHTML.RenderString(map[string]any{
		"R":R,
		"class":class,
		"postid":postid,
		"action":action,
	})
}

var ReactionHTMLComment = InlineUnsafeComponent(`
	<button
		title="{{ .R.Name }}" style="{{ .R.AltStyle }}" class="reaction {{ .class }}"
		{{ .action }}="/social/comments/react?reaction={{ .R.ID }}&commentid={{ .commentid }}"
		hx-target="#comment-{{ .commentid }} > span.reactions"
		hx-swap="outerHTML"
	>
		<img src="{{.R.Img}}" alt="{{.R.Alt}}">
	</button>
`)

func (R ReactionInfo) HTMLstrComment(selected uint64, commentid int64) (string) {
	if (R.ID == 0) {return ""}
	class:="notSelected"
	action:="hx-post"
	if selected == R.ID {
		class="selected"
		action="hx-delete"
	}
	return ReactionHTMLComment.RenderString(map[string]any{
		"R":R,
		"class":class,
		"commentid":commentid,
		"action":action,
	})
}

const (
	NoReaction = iota
	ReactionLike
	ReactionDislike
	ReactionLove
	ReactionHate
	ReactionLaugh
	ReactionCry
	ReactionLimit
)

var ReactionsInfo = [...]ReactionInfo{
	ReactionLike: ReactionInfo{ ReactionLike, "Like", true,
		"/files/img/social/reaction_like.ico", "↑", template.CSS("color: green;"),
	},
	ReactionDislike: ReactionInfo{ ReactionDislike, "Dislike", false,
		"/files/img/social/reaction_dislike.ico", "↓", template.CSS("color: red;"),
	},
	ReactionLove: ReactionInfo{ ReactionLove, "Love", true,
		"/files/img/social/reaction_love.ico", "<3", template.CSS("color: pink;"),
	},
	ReactionHate: ReactionInfo{ ReactionHate, "Hate", false,
		"/files/img/social/reaction_hate.ico", "`^´", template.CSS("color: red;"),
	},
	ReactionLaugh: ReactionInfo{ ReactionLaugh, "Laugh", true,
		"/files/img/social/reaction_laugh.ico", "XD", template.CSS("color: white;"),
	},
	ReactionCry: ReactionInfo{ ReactionCry, "Cry", false,
		"/files/img/social/reaction_cry.ico", ":(", template.CSS("color: blue;"),
	},
}

type Community struct {
	CommunityID int64
	Creator *Account
	Name string
	Description string
	Posts []*Post
	Subscount verfit.AUint64
}

type Post struct {
	PostID int64
	Poster *Account
	Community *Community
	PostText string
	PostHTML template.HTML // markdown -> html!
	PostTime time.Time
	Comments []*Comment
	CommentCount *verfit.AUint64
	LikeCount *verfit.AUint64
	DislikeCount *verfit.AUint64
	Reactions *SyncMap[int64, ReactionType]
	// ReactionCount for specific analytics
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
	ChildrenCount *verfit.AUint64
	LikeCount *verfit.AUint64
	DislikeCount *verfit.AUint64
	Reactions *SyncMap[int64, ReactionType]
	// ReactionCount for specific analytics
}

var CommentHTMLComponent = TemplatedComponent("html/social/components/comment.gohtml")

func (C Comment) HTMLR(viewer int64, depth int) template.HTML {
	return template.HTML(CommentHTMLComponent.RenderString( map[string]any{
		"comment":C,
		"depth": depth,
		"nextdepth": (depth%4)+1,
		"viewer":viewer,
		"allreactions":ReactionsInfo,
	}))
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
	_, e := SQLDo("service/social.(*Comment).React", `
	INSERT OR REPLACE INTO social_comment_reaction
		(commentID, accID, action)
	VALUES
		(?, ?, ?);`,
	C.CommentID, accID, reaction)
	if (e != nil) {panic(e)}
	oldr, change := C.Reactions.Get(accID)
	if (change) {
		if (ReactionsInfo[oldr].MeansHappy) {
			C.LikeCount.Add(^uint64(0))
		} else {
			C.DislikeCount.Add(^uint64(0))
		}
	}
	if (reaction == 0) {
		C.Reactions.Unset(accID)
	} else {
		C.Reactions.Set(accID, reaction)
		if (ReactionsInfo[reaction].MeansHappy) {
			C.LikeCount.Add(1)
		} else {
			C.DislikeCount.Add(1)
		}
	}
}

func (P *Post) React(accID int64, reaction ReactionType) {
	_, e := SQLDo("service/social.(*Post).React", `
	INSERT OR REPLACE INTO social_post_reaction
		(postID, accID, action)
	VALUES
		(?, ?, ?);`,
	P.PostID, accID, reaction)
	if (e != nil) {
		fmt.Println(e)
		panic(e)
	}
	oldr, change := P.Reactions.Get(accID)
	if (change) {
		if (ReactionsInfo[oldr].MeansHappy) {
			P.LikeCount.Add(^uint64(0))
		} else {
			P.DislikeCount.Add(^uint64(0))
		}
	}
	if (reaction == 0) {
		P.Reactions.Unset(accID)
	} else {
		P.Reactions.Set(accID, reaction)
		if (ReactionsInfo[reaction].MeansHappy) {
			P.LikeCount.Add(1)
		} else {
			P.DislikeCount.Add(1)
		}
	}
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
		verfit.AUint64{},
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

	// parentPost.Comments only accept root comments
	if (parentComment == nil) {
		parentPost.Comments = append(parentPost.Comments, c)
	}
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
		&verfit.AUint64{},
		&verfit.AUint64{},
		&verfit.AUint64{},
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
		&verfit.AUint64{},
		&verfit.AUint64{},
		&verfit.AUint64{},
		NewSyncMap[int64, ReactionType](),
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
		&verfit.AUint64{},
		&verfit.AUint64{},
		&verfit.AUint64{},
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
		atomicSubC := verfit.AUint64{}
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
		atomicCommentCount := verfit.AUint64{}
		atomicCommentCount.Store(commentCount)
		p := &Post{
			PostID, Poster, Comm,
			PostText,
			MDToHTML(PostText),
			PostTime,
			[]*Comment{},
			&atomicCommentCount,
			&verfit.AUint64{},
			&verfit.AUint64{},
			NewSyncMap[int64, ReactionType](),
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
		atomicChildrenCount := verfit.AUint64{}
		atomicChildrenCount.Store(childrenCount)
		c := &Comment{
			commentID, commenter, post,
			commentText,
			MDToHTML(commentText),
			commentTime,
			nil,
			[]*Comment{},
			&atomicChildrenCount,
			&verfit.AUint64{},
			&verfit.AUint64{},
			NewSyncMap[int64, ReactionType](),
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

