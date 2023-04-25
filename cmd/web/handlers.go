package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/CloudyKit/jet/v6"
	"github.com/go-chi/chi/v5"
	"github.com/nehulsukralia/newsWebApp/forms"
	"github.com/nehulsukralia/newsWebApp/models"
)

func (a *application) homeHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		a.serverError(w, err)
		return
	}

	filter := models.Filter{
		Query:    r.URL.Query().Get("q"),
		Page:     a.readIntDefault(r, "page", 1),
		PageSize: a.readIntDefault(r, "page_size", 5),
		OrderBy:  r.URL.Query().Get("order_by"),
	}

	posts, meta, err := a.Models.Posts.GetAll(filter)
	if err != nil {
		a.serverError(w, err)
		return
	}

	queryUrl := fmt.Sprintf("page_size=%d&order_by=%s&q=%s", meta.PageSize, filter.OrderBy, filter.Query)
	nextUrl := fmt.Sprintf("%s&page=%d", queryUrl, meta.NextPage)
	prevUrl := fmt.Sprintf("%s&page=%d", queryUrl, meta.PrevPage)

	vars := make(jet.VarMap)
	vars.Set("posts", posts)
	vars.Set("meta", meta)
	vars.Set("nextUrl", nextUrl)
	vars.Set("prevUrl", prevUrl)
	vars.Set("form", forms.New(r.Form))

	err = a.render(w, r, "index", vars)
	log.Println()
	if err != nil {
		log.Fatal(err)
	}
}

func (a *application) commentHandler(w http.ResponseWriter, r *http.Request) {
	vars := make(jet.VarMap)

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		a.clientError(w, http.StatusBadRequest)
		return
	}

	post, err := a.Models.Posts.Get(postId)
	if err != nil {
		a.serverError(w, err)
		return
	}

	comments, err := a.Models.Comments.GetForPost(postId)
	if err != nil {
		a.serverError(w, err)
		return
	}

	vars.Set("post", post)
	vars.Set("comments", comments)

	err = a.render(w, r, "comments", vars)
	if err != nil {
		a.serverError(w, err)
		return
	}
}

func (a *application) loginHandler(w http.ResponseWriter, r *http.Request) {
	err := a.render(w, r, "login", nil)
	if err != nil {
		a.serverError(w, err)
		return
	}
}

func (a *application) signUpHandler(w http.ResponseWriter, r *http.Request) {
	vars := make(jet.VarMap)
	vars.Set("form", forms.New(r.PostForm))

	err := a.render(w, r, "signup", vars)
	if err != nil {
		a.serverError(w, err)
		return
	}
}

func (a *application) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*2) // to limit the size of input data

	err := r.ParseForm()
	if err != nil {
		a.serverError(w, err)
		return
	}

	// validating form
	form := forms.New(r.PostForm)
	form.Email("email")
	form.MinLength("password", 3)

	if !form.Valid() {
		vars := make(jet.VarMap)
		vars.Set("errors", form.Errors)
		err := a.render(w, r, "login", vars)
		if err != nil {
			a.serverError(w, err)
			return
		}
	}

	// authenticating user
	user, err := a.Models.Users.Authenticate(form.Get("email"), form.Get("password"))
	if err != nil {
		a.session.Put(r.Context(), "flash", "Login error: "+err.Error())
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// putting login info in session
	a.session.RenewToken(r.Context()) // making new session
	a.session.Put(r.Context(), sessionKeyUserId, user.ID)
	a.session.Put(r.Context(), sessionKeyUserName, user.Name)
	http.Redirect(w, r, "/", http.StatusSeeOther)


}

func (a *application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// remove session data
	a.session.Remove(r.Context(), sessionKeyUserId)
	a.session.Remove(r.Context(), sessionKeyUserName)
	a.session.Destroy(r.Context())
	a.session.RenewToken(r.Context())

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}

func (a *application) signUpPostHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*2) // not allowing large body size

	err := r.ParseForm()
	if err != nil {
		a.serverError(w, err)
		return
	}

	vars := make(jet.VarMap)

	form := forms.New(r.PostForm)
	vars.Set("form", form)

	form.Required("name", "email", "password").
		Email("email")

	if !form.Valid() {
		vars.Set("errors", form.Errors)
		err := a.render(w, r, "signup", vars)
		if err != nil {
			a.serverError(w, err)
		}
		return
	}
	
	user := models.User{
		Name: form.Get("name"),
		Email: form.Get("email"),
		Password: form.Get("password"),
		Activated: true, // move to databse/config
	}
	err = a.Models.Users.Insert(&user)
	if err != nil {
		form.Fail("signup", "failed to create account: " + err.Error())
		vars.Set("errors", form.Errors)
		err := a.render(w, r, "signup", vars)
		if err != nil {
			a.serverError(w, err)
		}
		return
	}

	a.session.Put(r.Context(), "flash", "account created!")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *application) voteHandler(w http.ResponseWriter, r *http.Request) {
	id := a.readIntDefault(r, "id", 0)

	post, err := a.Models.Posts.Get(id)
	if err != nil {
		a.session.Put(r.Context(), "flash", "Error: "+err.Error())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userId := a.session.GetInt(r.Context(), sessionKeyUserId)
	err = a.Models.Posts.Vote(post.ID, userId)
	if err != nil {
		a.session.Put(r.Context(), "flash", "Error: "+err.Error())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	a.session.Put(r.Context(), "flash", "Voted successfully!")
}