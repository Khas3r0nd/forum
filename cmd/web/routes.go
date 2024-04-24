package main

import "net/http"

// The routes() method returns a servemux containing our application routes.
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	mux.HandleFunc("/", app.homePage)
	mux.HandleFunc("/post/create", app.postCreate)
	mux.HandleFunc("/user/signup", app.userSignup)
	mux.HandleFunc("/post/view", app.postView)
	mux.HandleFunc("/post/update", app.postUpdate)
	mux.HandleFunc("/commentEdit", app.commentUpdate)
	mux.HandleFunc("/user/login", app.userLogin)
	mux.HandleFunc("/user/logout", app.userLogout)
	mux.HandleFunc("/myactivities", app.userActivities)
	mux.HandleFunc("/myactivities/myposts", app.userPosts)
	mux.HandleFunc("/myactivities/likedposts", app.userLikedPosts)
	mux.HandleFunc("/myactivities/comments", app.userComments)
	mux.HandleFunc("/notifications", app.notifications)
	mux.HandleFunc("/authgoogle", app.googleLogin)
	mux.HandleFunc("/googlecallback", app.googleCallback)
	mux.HandleFunc("/authgithub", app.githubLogin)
	mux.HandleFunc("/githubcallback", app.githubCallback)
	return app.logRequest(secureHeaders(app.tokenBucketHandler(10, 10, mux)))
}
