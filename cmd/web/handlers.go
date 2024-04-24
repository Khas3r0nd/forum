package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"forum/internal/models"
	"forum/internal/validator"
	"forum/pkg"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

type postCreateForm struct {
	Title    string
	Content  string
	Created  time.Time
	UserId   string
	Category string
	Author   string
	Image    string
	validator.Validator
}

type userSignupForm struct {
	Name      string
	Email     string
	Password  string
	PasswordC string
	validator.Validator
}

type userLoginForm struct {
	UserID   int
	Email    string
	Password string
	validator.Validator
}

type commentForm struct {
	PostID  int
	Comment string
	validator.Validator
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		data := app.newTemplateData(r)
		data.Form = userSignupForm{}
		app.render(w, http.StatusOK, "signup.tmpl", data)
	case "POST":
		if err := r.ParseForm(); err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		form := userSignupForm{
			Name:      r.PostForm.Get("name"),
			Email:     r.PostForm.Get("email"),
			Password:  r.PostForm.Get("password"),
			PasswordC: r.PostForm.Get("passwordC"),
		}

		form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
		form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
		form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
		form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
		form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")
		form.CheckField(form.Password == form.PasswordC, "password", "Passwords must match")
		// If there are any errors, redisplay the signup form along with a 422
		if app.users.GetEmail(form.Email) == models.ErrDuplicateEmail {
			form.AddFieldError("email", "Email address is already in use")
		}
		if app.users.GetUsername(form.Name) == models.ErrDuplicateUsername {
			form.AddFieldError("name", "Username is already in use")
		}

		// status code.
		if !form.Valid() {
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.tmpl", data)
			return
		}

		hashedPassword, err := HashPassword(form.Password)
		if err != nil {
			log.Fatal(err)
		}
		app.users.Insert(form.Name, form.Email, hashedPassword)
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) homePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
		return
	}

	switch r.Method {
	case "GET":
		posts, err := app.posts.Latest()
		if err != nil {
			app.serverError(w, err)
			return
		}
		data := app.newTemplateData(r)
		data.AuthenticatedUser = app.authenticatedUser(r)
		data.Posts = posts

		app.render(w, http.StatusOK, "home.tmpl", data)

	case "POST":
		err := r.ParseForm()
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		categoryArray := r.Form["category"]
		posts, err := app.posts.LatestWithCategory(categoryArray)
		if err != nil {
			app.serverError(w, err)
			return
		}
		data := app.newTemplateData(r)
		data.AuthenticatedUser = app.authenticatedUser(r)
		data.Posts = posts
		app.render(w, http.StatusOK, "home.tmpl", data)

	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) postView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}
	likes, err := app.reactions.GetLikes(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}
	dislikes, err := app.reactions.GetDislikes(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}
	switch r.Method {
	case "GET":
		post, err := app.posts.Get(id)
		if err != nil {
			if errors.Is(err, models.ErrNoRecord) {
				app.notFound(w)
			} else {
				app.serverError(w, err)
			}
			return
		}
		data := app.newTemplateData(r)
		data.Post = post
		data.Post.Likes = likes
		data.Post.Dislikes = dislikes
		filename, fileType, err := app.imageUpload.GetImage(id)
		if err != nil {
			fmt.Println("Error uploading image")
			return
		}
		data.Post.Image = fmt.Sprintf("/static/img/%s.%s", filename, fileType)
		data.Comments, err = app.comments.GetComments(id)

		for _, comment := range data.Comments {
			data.Comment = comment
			data.AuthenticatedUser = app.authenticatedUser(r)
			if data.AuthenticatedUser != 0 {
				cookie, err := r.Cookie("session_token")
				if err != nil {
					if err == http.ErrNoCookie {
						comment.CurrentUser = 0
					}
					if err == cookie.Valid() {
						return
					}
				}
				comment.CurrentUser = int(app.sessionManager.GetUserIDBySessionToken(cookie.Value))
				fmt.Println(comment.CurrentUser)
			}

			Commentlikes, err := app.reactions.GetCommentLikes(comment.CommentID)
			if err != nil {
				if errors.Is(err, models.ErrNoRecord) {
					app.notFound(w)
				} else {
					app.serverError(w, err)
				}
				return
			}
			Commentdislikes, err := app.reactions.GetCommentDislikes(comment.CommentID)
			if err != nil {
				if errors.Is(err, models.ErrNoRecord) {
					app.notFound(w)
				} else {
					app.serverError(w, err)
				}
				return
			}
			comment.CommentLikes = Commentlikes
			comment.CommentDislikes = Commentdislikes

		}
		data.Post.Categories, _ = app.postTags.GetCategoriesByPostID(id)
		if err != nil {
			app.serverError(w, err)
			return
		}
		data.AuthenticatedUser = app.authenticatedUser(r)
		if data.AuthenticatedUser != 0 {
			cookie, err := r.Cookie("session_token")
			if err != nil {
				if err == http.ErrNoCookie {
					return
				}
				if err == cookie.Valid() {
					return
				}
			}
			UserID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
			data.Post.Currentuser = UserID
			data.Post.UserId, err = app.posts.GetUserID(id)
		}

		data.Form = commentForm{}
		app.render(w, http.StatusOK, "view.tmpl", data)

	case "POST":
		if app.authenticatedUser(r) == 0 {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}
		if err := r.ParseForm(); err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		post, err := app.posts.Get(id)
		if err != nil {
			if errors.Is(err, models.ErrNoRecord) {
				app.notFound(w)
			} else {
				app.serverError(w, err)
			}
			return
		}
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				return
			}
			if err == cookie.Valid() {
				return
			}
		}
		UserID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
		data := app.newTemplateData(r)
		if r.Form.Has("Like") {
			switch r.PostForm.Get("Like") {
			case "1":
				app.reactions.MakeReaction(UserID, id, 1)
			case "-1":
				app.reactions.MakeReaction(UserID, id, -1)

			}
		}
		if r.Form.Has("comment") {
			form := commentForm{
				Comment: r.PostForm.Get("comment"),
			}
			form.CheckField(validator.NotBlank(form.Comment), "comment", "This field cannot be blank")
			form.CheckField(validator.MaxChars(form.Comment, 50), "comment", "This field cannot be more than 50 characters long")
			if !form.Valid() {
				data := app.newTemplateData(r)
				data.Form = form
				data.AuthenticatedUser = app.authenticatedUser(r)
				data.Post = post
				data.Comments, err = app.comments.GetComments(id)
				if err != nil {
					app.clientError(w, http.StatusBadRequest)
					return
				}

				data.Post.Likes = likes
				data.Post.Dislikes = dislikes
				app.render(w, http.StatusUnprocessableEntity, "view.tmpl", data)
				return
			}
			app.comments.Insert(UserID, id, form.Comment)

		}
		if r.Form.Has("LikeComment") {
			data.AuthenticatedUser = app.authenticatedUser(r)
			data.Comments, err = app.comments.GetComments(id)
			CommentID := r.PostForm.Get("LikeComment")
			CommentIDINT, err := strconv.Atoi(CommentID)
			if err != nil {
				return
			}
			for _, comment := range data.Comments {
				data.Comment = comment
				switch r.PostForm.Get("ReactComment") {
				case "1":
					if CommentIDINT == data.Comment.CommentID {
						app.reactions.MakeCommentReaction(UserID, data.Comment.CommentID, 1)
					}
				case "-1":
					if CommentIDINT == data.Comment.CommentID {
						app.reactions.MakeCommentReaction(UserID, data.Comment.CommentID, -1)
					}
				}
			}
		}
		if r.Form.Has("Delete") {
			err = app.posts.DeletePost(id, UserID)
			if err != nil {
				log.Println("ERROR", err)
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/"), 302)
			return
		}
		if r.Form.Has("Edit") {
			http.Redirect(w, r, fmt.Sprintf("/post/update?id=%d", id), http.StatusSeeOther)
			return
		}
		if r.FormValue("ChangeComment") != "" {
			fmt.Println("test")
			CommentID := r.PostForm.Get("EditComment")
			CommentIDINT, err := strconv.Atoi(CommentID)
			if err != nil {
				log.Println("ERROR", err)
				return
			}
			fmt.Println("ZAHODIT???")
			http.Redirect(w, r, fmt.Sprintf("/commentEdit?id=%d", CommentIDINT), http.StatusSeeOther)
			return
		} else if r.FormValue("DeleteComment") != "" {
			fmt.Println("I was here")
			CommentID := r.PostForm.Get("DeleteComment")
			CommentIDINT, err := strconv.Atoi(CommentID)
			fmt.Println(CommentIDINT)
			if err != nil {
				fmt.Println("ERROR", err)
				return
			}
			err = app.comments.DeleteComment(CommentIDINT, UserID)
			if err != nil {
				log.Println("ERROR", err)
				return
			}

			http.Redirect(w, r, fmt.Sprintf("/post/view?id=%d", id), 302)
			return
		}
		data.AuthenticatedUser = app.authenticatedUser(r)
		data.Post = post
		data.Comments, err = app.comments.GetComments(id)
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		data.Post.Likes = likes
		data.Post.Dislikes = dislikes
		http.Redirect(w, r, fmt.Sprintf("/post/view?id=%d", id), 302)
		return
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) postUpdate(w http.ResponseWriter, r *http.Request) {
	if app.authenticatedUser(r) == 0 {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			return
		}
		if err == cookie.Valid() {
			return
		}
	}
	UserID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
	switch r.Method {
	case "GET":
		post, err := app.posts.Get(id)
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		data := app.newTemplateData(r)
		data.Post = post
		data.Post.UserId, err = app.posts.GetUserID(id)
		fmt.Println(data.Post.UserId, "zzz")
		data.Post.Currentuser = UserID
		fmt.Println(data.Post.Currentuser, "aaa")
		data.AuthenticatedUser = app.authenticatedUser(r)
		app.render(w, http.StatusOK, "update.tmpl", data)
	case "POST":
		err := r.ParseForm()
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		Title := r.Form.Get("title")
		Content := r.Form.Get("content")

		err = app.posts.UpdatePost(Title, Content, id, UserID)
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/post/view?id=%d", id), 302)
		return
	}
}

func (app *application) commentUpdate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	if app.authenticatedUser(r) == 0 {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	fmt.Println("ID = ", id)
	if err != nil || id < 1 {
		fmt.Println("HOWWW", err)
		app.notFound(w)
		return
	}
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			return
		}
		if err == cookie.Valid() {
			return
		}
	}
	UserID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
	switch r.Method {
	case "GET":
		fmt.Println("GDE?! 111")
		comment, err := app.comments.GetCommentID(id)
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		data.Comment = comment
		data.Comment.CommentID = id
		fmt.Println(data.Comment.CommentID, "END?")
		data.AuthenticatedUser = app.authenticatedUser(r)
		app.render(w, http.StatusOK, "commentEdit.tmpl", data)
	case "POST":
		data := app.newTemplateData(r)
		comment, err := app.comments.GetCommentID(id)
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		fmt.Println(comment, "CHTO ZDES?")
		data.Comment = comment

		fmt.Println("GDE?! 222")
		err = r.ParseForm()
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		comments := r.Form.Get("comment")
		fmt.Println(comments, id, UserID)
		err = app.comments.EditComment(comments, id, UserID)
		if err != nil {
			fmt.Println("Error", err)
			return
		}

		post_id, err := app.comments.GetPostIDByCommentID(id)
		if err != nil {
			fmt.Println("Error", err)
			return
		}
		fmt.Println("POST-ID", post_id)
		http.Redirect(w, r, fmt.Sprintf("/post/view?id=%d", post_id), 302)
		return
	}
}

func (app *application) postCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	if app.authenticatedUser(r) == 0 {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}
	switch r.Method {
	case "GET":
		data.Form = postCreateForm{}
		data.AuthenticatedUser = app.authenticatedUser(r)
		app.render(w, http.StatusOK, "create.tmpl", data)
	case "POST":

		err := r.ParseForm()
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
		err = r.ParseMultipartForm(20 << 20)
		if err != nil {
			if !(err.Error() == "http: request body too large") {
				app.clientError(w, http.StatusInternalServerError)
				return
			}
		}
		form := postCreateForm{
			Title:    r.PostForm.Get("title"),
			Content:  r.PostForm.Get("content"),
			Category: r.PostForm.Get("category"),
			Image:    r.PostForm.Get("image"),
		}

		form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
		form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 characters long")
		form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
		form.CheckField(validator.NotBlank(form.Category), "category", "This field cannot be blank")
		form.CheckField(validator.ImageLimit(w, r), "image", "Maximum size image 20 Mb")
		if !form.Valid() {
			data := app.newTemplateData(r)
			data.Form = form
			data.AuthenticatedUser = app.authenticatedUser(r)
			app.render(w, http.StatusBadRequest, "create.tmpl", data)
			return
		}
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				return
			}
			if err == cookie.Valid() {
				return
			}
		}
		var filename, fileExt string
		file, header, err := r.FormFile("image")
		if err != nil {
			filename = "NoImage"
			fileExt = "jpg"
		} else {
			defer file.Close()
			buffer := make([]byte, 512)
			_, err = file.Read(buffer)
			if err != nil {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				log.Println("Failed to read file:", err)
				return
			}

			// Получение MIME-типа загруженного файла
			fileType := http.DetectContentType(buffer)
			if !strings.HasPrefix(fileType, "image/") {
				form.CheckField(validator.ImageExtension(fileType), "image", "NEED ONLY Image extensions")
			} else {
				_, err = file.Seek(0, io.SeekStart)
				if err != nil {
					http.Error(w, "Failed to rewind file", http.StatusInternalServerError)
					log.Println("Failed to rewind file:", err)
					return
				}
				uniqueId, err := uuid.NewV4()
				if err != nil {
					return
				}

				filename = strings.Replace(uniqueId.String(), "-", "", -1)
				fileExt = strings.Split(header.Filename, ".")[1]
				newFile, err := os.Create(fmt.Sprintf("./ui/static/img/%s.%s", filename, fileExt))
				if err != nil {
					http.Error(w, "Failed to create the file", http.StatusInternalServerError)
					log.Println("Failed to create the file:", err)
					return
				}
				defer newFile.Close()
				_, err = io.Copy(newFile, file)
				if err != nil {
					http.Error(w, "Failed to copy file content", http.StatusInternalServerError)
					log.Println("Failed to copy file content:", err)
					return
				}
			}
		}

		// Получение значения массива строк из формы
		values := r.Form["category"]
		form.Category = strings.Join(values, ", ")
		categories := strings.Split(form.Category, ", ")
		form.CheckField(validator.CheckCategory(categories), "category", "category is not right")

		if !form.Valid() {
			data := app.newTemplateData(r)
			data.Form = form
			data.AuthenticatedUser = app.authenticatedUser(r)
			app.render(w, http.StatusBadRequest, "create.tmpl", data)
			return
		}

		UserID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
		id, err := app.posts.Insert(form.Title, form.Content, UserID)
		if err != nil {
			app.serverError(w, err)
			return
		}
		err = app.imageUpload.InsertImage(id, filename, fileExt)
		if err != nil {
			fmt.Println("ERROR")
			return
		}

		for _, category := range categories {
			err := app.postTags.InsertCategory(id, category)
			if err != nil {
				app.serverError(w, err)
				return
			}
		}
		http.Redirect(w, r, fmt.Sprintf("/post/view?id=%d", id), http.StatusSeeOther)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := app.newTemplateData(r)
		data.Form = userLoginForm{}
		app.render(w, http.StatusOK, "login.tmpl", data)
	case "POST":
		if err := r.ParseForm(); err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		form := userLoginForm{
			Email:    r.PostForm.Get("email"),
			Password: r.PostForm.Get("password"),
		}

		form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
		form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
		form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
		if !form.Valid() {
			data := app.newTemplateData(r)
			data.Form = form
			data.AuthenticatedUser = app.authenticatedUser(r)
			app.render(w, http.StatusBadRequest, "login.tmpl", data)
			return
		}
		// Check whether the credentials are valid. If they're not, add a generic
		// non-field error message and re-display the login page.
		id, err := app.users.Authenticate(form.Email, form.Password)
		if err != nil {
			if errors.Is(err, models.ErrInvalidCredentials) {
				form.AddNonFieldError("Email or password is incorrect")
				data := app.newTemplateData(r)
				data.Form = form
				app.render(w, http.StatusUnprocessableEntity, "login.tmpl", data)
			} else {
				app.serverError(w, err)
			}
			return
		}
		app.sessionManager.CreateSession(w, r, id)
		// Получаем значение UserID из контекста

		http.Redirect(w, r, "/post/create", http.StatusSeeOther)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) userLogout(w http.ResponseWriter, r *http.Request) {
	// Получите сессию пользователя, если она существует
	sessionCookie, err := r.Cookie("session_token")

	if err == nil {
		// Удаление сессии
		err := app.sessionManager.DeleteSession(sessionCookie.Value)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}

	// Очистка куков и перенаправление на главную страницу
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Unix(0, 0),
		Path:    "/",
		MaxAge:  -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) userPosts(w http.ResponseWriter, r *http.Request) {
	if app.authenticatedUser(r) == 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case "GET":
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				return
			}
			if err == cookie.Valid() {
				return
			}
		}
		userID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
		posts, err := app.posts.GetUserPosts(userID)
		if err != nil {
			app.serverError(w, err)
			return
		}

		data := app.newTemplateData(r)
		data.AuthenticatedUser = app.authenticatedUser(r)
		data.Posts = posts
		app.render(w, http.StatusOK, "userPosts.tmpl", data)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}
}

func (app *application) userLikedPosts(w http.ResponseWriter, r *http.Request) {
	if app.authenticatedUser(r) == 0 {
		app.clientError(w, http.StatusNotFound)
		return
	}
	switch r.Method {
	case "GET":
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				return
			}
			if err == cookie.Valid() {
				return
			}
		}
		userID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
		posts, err := app.posts.GetUserLikedPosts(userID)
		if err != nil {
			app.serverError(w, err)
			return
		}
		data := app.newTemplateData(r)
		data.AuthenticatedUser = app.authenticatedUser(r)
		data.Posts = posts
		app.render(w, http.StatusOK, "userLiked.tmpl", data)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) notifications(w http.ResponseWriter, r *http.Request) {
	if app.authenticatedUser(r) == 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case "GET":
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				return
			}
			if err == cookie.Valid() {
				return
			}
		}
		userID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
		data := app.newTemplateData(r)
		data.Comments, err = app.comments.CommentNotifications(userID)
		for _, comment := range data.Comments {
			data.Comment = comment
			fmt.Println(comment.UserID)

			fmt.Println(comment.Username)
		}
		data.Reactions, err = app.reactions.GetPostReactions(userID)
		for _, reaction := range data.Reactions {
			data.Reaction = reaction
			fmt.Println(reaction.Username, "123456789")
			fmt.Println(reaction.LikeStatus, "++++")
		}
		data.AuthenticatedUser = app.authenticatedUser(r)
		app.render(w, http.StatusOK, "notifications.tmpl", data)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) userActivities(w http.ResponseWriter, r *http.Request) {
	if app.authenticatedUser(r) == 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case "GET":
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				return
			}
			if err == cookie.Valid() {
				return
			}
		}
		data := app.newTemplateData(r)
		data.AuthenticatedUser = app.authenticatedUser(r)
		app.render(w, http.StatusOK, "userActivities.tmpl", data)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) userComments(w http.ResponseWriter, r *http.Request) {
	if app.authenticatedUser(r) == 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case "GET":
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				return
			}
			if err == cookie.Valid() {
				return
			}
		}
		userID := app.sessionManager.GetUserIDBySessionToken(cookie.Value)
		data := app.newTemplateData(r)
		data.Comments, err = app.comments.GetUserComments(userID)
		for _, comment := range data.Comments {
			data.Comment = comment
		}
		data.AuthenticatedUser = app.authenticatedUser(r)

		app.render(w, http.StatusOK, "userComments.tmpl", data)
	default:
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func saveFile(fileHeader *multipart.FileHeader, destPath string) error {
	source, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func (app *application) googleLogin(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&prompt=select_account",
		models.GoogleAuthURL, models.GoogleClientID, models.GoogleRedirectUrl, "email profile")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (app *application) googleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	pathURL := "/"

	state := r.FormValue("state")
	if state != "" {
		pathURL = state
	}

	if code == "" {
		app.clientError(w, http.StatusInternalServerError)
		return
	}

	tokenRes, err := GetGoogleOauthToken(code)
	if err != nil {

		app.clientError(w, http.StatusInternalServerError)
		return
	}

	googleUser, err := GetGoogleUser(tokenRes.Access_token, tokenRes.Id_token)
	if err != nil {

		app.clientError(w, http.StatusInternalServerError)
		return
	}

	exists, err := app.users.DoesUserExistByEmail(googleUser.Email)

	if err != nil {

		app.clientError(w, http.StatusInternalServerError)
		return
	} else if !exists {

		password := pkg.GetToken()
		app.users.Insert(googleUser.Name, googleUser.Email, password)
		id, err := app.users.ReturnUserID(googleUser.Email)
		if err != nil {

			app.clientError(w, http.StatusInternalServerError)
			return
		}

		err = app.sessionManager.CreateSession(w, r, id)
		if err != nil {

			app.clientError(w, http.StatusInternalServerError)
			return

		}

	} else {

		id, err := app.users.ReturnUserID(googleUser.Email)
		if err != nil {

			fmt.Println(err)
			app.clientError(w, http.StatusInternalServerError)
			return
		}

		err = app.sessionManager.CreateSession(w, r, id)
	}

	http.Redirect(w, r, pathURL, http.StatusTemporaryRedirect)
}

type GoogleUserResult struct {
	Id             string
	Email          string
	Verified_email bool
	Name           string
	Given_name     string
	Family_name    string
	Picture        string
	Locale         string
}

func GetGoogleUser(access_token string, id_token string) (*GoogleUserResult, error) {
	rootUrl := fmt.Sprintf("https://www.googleapis.com/oauth2/v1/userinfo?alt=json&access_token=%s", access_token)

	req, err := http.NewRequest("GET", rootUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", id_token))

	client := http.Client{
		Timeout: time.Second * 30,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("could not retrieve user")
	}

	var resBody bytes.Buffer

	if _, err = io.Copy(&resBody, res.Body); err != nil {
		return nil, err
	}
	var GoogleUserRes map[string]interface{}

	if err := json.Unmarshal(resBody.Bytes(), &GoogleUserRes); err != nil {
		return nil, err
	}

	userBody := &GoogleUserResult{
		Id:             GoogleUserRes["id"].(string),
		Email:          GoogleUserRes["email"].(string),
		Verified_email: GoogleUserRes["verified_email"].(bool),
		Name:           GoogleUserRes["name"].(string),
		Given_name:     GoogleUserRes["given_name"].(string),
		Picture:        GoogleUserRes["picture"].(string),
		Locale:         GoogleUserRes["locale"].(string),
	}
	return userBody, nil
}

type GoogleOauthToken struct {
	Access_token string
	Id_token     string
}

func GetGoogleOauthToken(code string) (*GoogleOauthToken, error) {
	const rootURl = "https://oauth2.googleapis.com/token"

	values := url.Values{}
	values.Add("grant_type", "authorization_code")
	values.Add("code", code)
	values.Add("client_id", models.GoogleClientID)
	values.Add("client_secret", models.GoogleClientSecret)
	values.Add("redirect_uri", models.GoogleRedirectUrl)

	query := values.Encode()

	req, err := http.NewRequest("POST", rootURl, bytes.NewBufferString(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{
		Timeout: time.Second * 30,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("could not retrieve token")
	}

	var resBody bytes.Buffer
	_, err = io.Copy(&resBody, res.Body)
	if err != nil {
		return nil, err
	}

	var GoogleOauthTokenRes map[string]interface{}

	if err := json.Unmarshal(resBody.Bytes(), &GoogleOauthTokenRes); err != nil {
		return nil, err
	}

	tokenBody := &GoogleOauthToken{
		Access_token: GoogleOauthTokenRes["access_token"].(string),
		Id_token:     GoogleOauthTokenRes["id_token"].(string),
	}
	return tokenBody, nil
}

//////////////

func (app *application) githubLogin(w http.ResponseWriter, r *http.Request) {
	redirectURL := fmt.Sprintf(
		"%ss?client_id=%s&redirect_uri=%s",
		models.GitHubAuthURL,
		models.GithubClientID,
		models.GithubRedirectUrl,
	)

	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func (app *application) githubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	githubAccessToken, err := getGithubAccessToken(code)
	if err != nil {
		app.clientError(w, http.StatusInternalServerError)
		return
	}

	githubData, err := getGithubData(githubAccessToken)
	if err != nil {
		app.clientError(w, http.StatusInternalServerError)
		return
	}

	userData, err := getUserData(githubData)
	if err != nil {
		app.clientError(w, http.StatusInternalServerError)
		return
	}

	exists, err := app.users.DoesUserExistByName(userData.UserName)
	if err != nil {
		fmt.Println(err)
		app.clientError(w, http.StatusInternalServerError)
		return
	}
	if !exists {
		password := pkg.GetToken()

		app.users.Insert(userData.UserName, userData.UserName+"@github.com", password)
		id, err := app.users.ReturnUserID(userData.UserName + "@github.com")
		if err != nil {
			app.clientError(w, http.StatusInternalServerError)
			return
		}
		app.sessionManager.CreateSession(w, r, id)
	} else {
		id, err := app.users.ReturnUserID(userData.UserName + "@github.com")
		if err != nil {
			app.clientError(w, http.StatusInternalServerError)
			return
		}
		app.sessionManager.CreateSession(w, r, id)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func getUserData(data string) (models.GithubLoginUserData, error) {
	userData := models.GithubLoginUserData{}
	if err := json.Unmarshal([]byte(data), &userData); err != nil {
		return models.GithubLoginUserData{}, err
	}

	return userData, nil
}

func getGithubAccessToken(code string) (string, error) {
	requestBodyMap := map[string]string{
		"client_id":     models.GithubClientID,
		"client_secret": models.GithubClientSecret,
		"code":          code,
	}
	requestJSON, err := json.Marshal(requestBodyMap)
	if err != nil {
		return "", err
	}

	req, reqerr := http.NewRequest(
		"POST",
		"https://github.com/login/oauth/access_token",
		bytes.NewBuffer(requestJSON),
	)
	if reqerr != nil {
		return "", reqerr
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		return "", resperr
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type githubAccessTokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	var ghresp githubAccessTokenResponse
	if err := json.Unmarshal(respbody, &ghresp); err != nil {
		return "", err
	}

	return ghresp.AccessToken, nil
}

func getGithubData(accessToken string) (string, error) {
	req, reqerr := http.NewRequest(
		"GET",
		"https://api.github.com/user",
		nil,
	)
	if reqerr != nil {
		return "", reqerr
	}

	authorizationHeaderValue := fmt.Sprintf("token %s", accessToken)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		return "", resperr
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respbody), nil
}
