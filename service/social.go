package service

import (
	. "mysrv/util"
	"time"
	"fmt"
	"database/sql"
	"sync/atomic"
	"html/template"
	"github.com/gomarkdown/markdown"
	"strconv"
	"io"
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

var AllEndpoint = LogicPage(
	"html/social/cards.gohtml",
	map[string]any{
		"allreactions":ReactionsInfo,
	},
	[]GOTMPlugin{
		GOTM_account,
		GOTM_mustacc,
	},
	postsEndpoint,
)

type SocialQuery struct {
	AccID int64
	CommunityID int64
	PostID int64
	CommentID int64
	Reaction ReactionType
	PostCount int64
	UseSort SortMethod
}

func prelude(w HttpWriter, r HttpReq, info map[string]any) (acc *Account, query SocialQuery, ok bool) {
	r.ParseForm()
	accid := (info["acc"].(map[string]any))["id"].(int64)
	query.AccID = accid
	acc, ok = IDToAccount.Get(accid)
	if (!ok) {return}
	query.CommunityID, _ = strconv.ParseInt(r.FormValue("communityid"), 10, 64)
	query.PostID, _      = strconv.ParseInt(r.FormValue("postid"), 10, 64)
	query.CommentID, _   = strconv.ParseInt(r.FormValue("commentid"), 10, 64)
	query.Reaction, _    = strconv.ParseUint(r.FormValue("reaction"), 10, 64)
	query.PostCount, _    = strconv.ParseInt(r.FormValue("postcount"), 10, 64)
	query.UseSort = SortNameToID.GetI(r.FormValue("sortmethod"))
	if (query.PostCount == 0) {
		query.PostCount = 10
	}
	return
}

func postsEndpoint(w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any) {
	_, query, ok := prelude(w, r, info)
	if (!ok) {return}

	var sortedPosts = _AllSorted[query.UseSort]
	pcount := Min(int(query.PostCount), len(sortedPosts))
	torender := sortedPosts[:pcount]
	//var i int64 = 0
	// pre-sort by time; gather only PostCount
	//for post := range IDToPost.IterValues() {
	//	torender = append(torender, post)
	//	i++
	//}
	return true, map[string]any{
		"query": query,
		"posts": torender,
	}

	//sp := SortedPosts{torender, query.UseSort}
	//sort.Sort(sp)
	//if (int64(len(torender)) > query.PostCount) {
	//	torender = torender[:query.PostCount]
	//}

	//return true, map[string]any{
	//	"query": query,
	//	"posts": torender,
	//}
}


var PostPageTemplate = template.Must(template.New("Post.Page").Parse(`
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
`))

func (P *Post) HTMLPage(w io.Writer, viewer *Account) {
	e := PostPageTemplate.Execute(w, map[string]any{
	"allreactions":ReactionsInfo,
	"post":*P,
	"acc":viewer,
})
	if (e!=nil) { panic(e) }
}

func MDToHTML(MD string) (HTML template.HTML) {
	return template.HTML(string(markdown.ToHTML([]byte(MD), nil, nil)))
}

var (
	_AllSorted = [_TotalSortMethods][]*Post{
		[]*Post{}, []*Post{}, []*Post{},
		[]*Post{}, []*Post{}, []*Post{},
		[]*Post{}, []*Post{}, []*Post{},
	}
	SortedPosts_Newest = &_AllSorted[0]
	SortedPosts_Oldest = &_AllSorted[1]
	SortedPosts_Comments = &_AllSorted[2]
	SortedPosts_Likes = &_AllSorted[3]
	SortedPosts_Dislikes = &_AllSorted[4]
	SortedPosts_Divided = &_AllSorted[5]
	SortedPosts_United = &_AllSorted[6]
	SortedPosts_UnitedLove = &_AllSorted[7]
	SortedPosts_UnitedHate = &_AllSorted[8]
)

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
	_TotalSortMethods = iota-1 // 10
)

type SortSpot [_TotalSortMethods]int
type SortCompares [_TotalSortMethods]bool

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
var SortIDToName = RevertMap(SortNameToID.AMap())

func PlaceInSorted(newpost *Post) {
	sspt := FindSpots(newpost)
	for SortType, spot := range sspt {
		if (spot == 0) {
			_AllSorted[SortType] = append( []*Post{newpost}, _AllSorted[SortType]...)
		} else if (spot == len(_AllSorted[SortType])) {
			_AllSorted[SortType] = append( _AllSorted[SortType], newpost)
		} else {
			_AllSorted[SortType] = append(
				append(_AllSorted[SortType][:spot], newpost),
				_AllSorted[SortType][spot:]...,
			)
		}
	}
	newpost.SortingSpots = sspt
}

//func UpdateStop(sspt SortSpot)

func FindSpots(newpost *Post) (sspt SortSpot) {
	var scmp SortCompares
	var sptFound SortCompares
	for i:=0;i<len(_AllSorted[_TotalSortMethods-1]);i++ {
		scmp = ComparePosts(newpost, i)
		for SortType, IsBefore := range scmp {
			if (IsBefore && !sptFound[SortType]) {
				sptFound[SortType] = true
				sspt[SortType] = i
			}
		}
	}
	for i:=0;i<_TotalSortMethods;i++ {
		if (!sptFound[i]) {
			sspt[i] = len(_AllSorted[_TotalSortMethods-1])
		}
	}
	return
}

// check if PostA is more important than PostB
func ComparePosts(PostA *Post, PostBIndex int) (scmp SortCompares) {
	//assert PostA && PostB != nil

	//TODO get united/diveded ratio
	SPosts := [_TotalSortMethods]*Post {
		(*SortedPosts_Newest    )[PostBIndex],
		(*SortedPosts_Oldest    )[PostBIndex],
		(*SortedPosts_Comments  )[PostBIndex],
		(*SortedPosts_Likes     )[PostBIndex],
		(*SortedPosts_Dislikes  )[PostBIndex],
		(*SortedPosts_Divided   )[PostBIndex],
		(*SortedPosts_United    )[PostBIndex],
		(*SortedPosts_UnitedLove)[PostBIndex],
		(*SortedPosts_UnitedHate)[PostBIndex],
	}
	scmp[0] = PostA.PostTime.After      ( SPosts[0].PostTime )
	scmp[1] = PostA.PostTime.Before     ( SPosts[1].PostTime )
	scmp[2] = PostA.CommentCount.Load() > SPosts[2].CommentCount.Load()
	scmp[3] = PostA.LikeCount.Load()    > SPosts[3].LikeCount.Load()
	scmp[4] = PostA.DislikeCount.Load() > SPosts[4].DislikeCount.Load()
	scmp[5] = PostA.DislikeCount.Load() > SPosts[5].DislikeCount.Load()
	scmp[6] = PostA.DislikeCount.Load() > SPosts[6].DislikeCount.Load()
	scmp[7] = PostA.DislikeCount.Load() > SPosts[7].DislikeCount.Load()
	scmp[8] = PostA.DislikeCount.Load() > SPosts[8].DislikeCount.Load()
	return
}

func DebugSocial() {
	for SortID, plist := range _AllSorted {
		fmt.Printf("%s:%v\n", SortIDToName[int64(SortID+1)], plist)
	}
	//op := EmailToAccount.GetI("pedro@manse.dev")
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
	ReactionCry
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
	ReactionCry: ReactionInfo{ ReactionCry, "Cry", false,
	"/files/img/reaction_cry", ":(", "color: blue;",
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
	SortingSpots SortSpot
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

// listeners
var (
	LoadPostEvent Event[*Post] // _loadposts
	NewPostEvent Event[*Post] // createPost
	InstancePostEvent Event[*Post] // _loadposts || createPost
)

func InstancePost(P *Post) (stopListener bool) {
	IDToPost.Set(P.PostID, P)
	P.Community.Posts = append(P.Community.Posts, P)
	PlaceInSorted(P)
	return
}

func init() {
	SQLInitScript( "social#create tables", socialSQLTables )
	SQLInitFunc( "social#load", loadSQL )

	InstancePostEvent.Listen(InstancePost)
	IDToPost.Init()
	IDToComment.Init()
	IDToCommunity.Init()
	UIDToSubs.Init()
	CIDToSubs.Init()
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
	// instance post
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
		SortSpot{},
	}
	NewPostEvent.Alert(p)
	InstancePostEvent.Alert(p)

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
		// instance post
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
			SortSpot{},
		}
		LoadPostEvent.Alert(p)
		InstancePostEvent.Alert(p)

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

