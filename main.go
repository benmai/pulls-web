package main

import (
	"code.google.com/p/goauth2/oauth"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	port    = flag.String("p", "8000", "Port number (default 8000)")
	orgName = flag.String("org", "", "GitHub organization name")
)

type Repository struct {
	github.Repository
	PullRequests []github.PullRequest
}

func getPullsForRepo(wg *sync.WaitGroup, c *github.Client, gitHubRepo github.Repository, repos []Repository, i int) {
	defer wg.Done()
	pulls, _, err := c.PullRequests.List(*orgName, *gitHubRepo.Name, nil)
	if err != nil {
		log.Panic(err)
	}
	repo := Repository{}
	repo.Repository = gitHubRepo
	repo.PullRequests = pulls
	repos[i] = repo
}

func getRepos() []Repository {
	var wg sync.WaitGroup
	page := 0
	githubToken := os.Getenv("GITHUB_API_TOKEN")
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: githubToken},
	}
	client := github.NewClient(t.Client())
	gitHubRepos, _, err := client.Repositories.ListByOrg(*orgName, nil)
	allGitHubRepos := []github.Repository{}
	if err != nil {
		log.Panic(err)
	}
getAllRepos:
	if len(gitHubRepos) < 30 {
		allGitHubRepos = append(allGitHubRepos, gitHubRepos...)
	} else {
		page = page + 1
		opt := &github.RepositoryListByOrgOptions{"", github.ListOptions{Page: page}}
		gitHubRepos, _, err = client.Repositories.ListByOrg(*orgName, opt)
		if err != nil {
			println(err)
		}
		allGitHubRepos = append(allGitHubRepos, gitHubRepos...)
		goto getAllRepos
	}
	repos := make([]Repository, len(allGitHubRepos))
	for i, repo := range allGitHubRepos {
		wg.Add(1)
		go getPullsForRepo(&wg, client, repo, repos, i)
	}
	wg.Wait()
	return repos
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	repos := getRepos()
	tmplRepos := []Repository{}
	for _, repo := range repos {
		if len(repo.PullRequests) > 0 {
			tmplRepos = append(tmplRepos, repo)
		}
	}
	t, err := template.New("index.html").ParseFiles("templates/index.html")
	if err != nil {
		log.Panic(err)
	}
	// Render the template
	err = t.Execute(w, map[string]interface{}{"Repos": tmplRepos})
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/", HomeHandler)
	fmt.Println("Running on localhost:" + *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
