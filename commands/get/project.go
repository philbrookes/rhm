package get

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"text/template"

	"github.com/feedhenry/rhm/commands"
	"github.com/feedhenry/rhm/storage"
	"github.com/urfave/cli"
)

var projectTemplate = `
| Title |  {{.Title}} 
| Email |  {{.AuthorEmail}} 
| Guid  |  {{.GUID}}
| Type  |  {{.Type}}
        {{if .Apps}}
        | Apps |
                {{range .Apps }}
               | Title | {{.Title}}
               | Guid  | {{.GUID}}
            
                {{end}}
        {{end}}        
`

type projectCmd struct {
	in      io.Reader
	out     io.Writer
	store   storage.Storer
	project string
	getter  func(*http.Request) (*http.Response, error)
}

func (pc *projectCmd) Project() cli.Command {
	return cli.Command{
		Name:        "project",
		Action:      pc.projectAction,
		Description: "get the project definition. If you have set rhm use project <guid> it will use that project. Or you can pass --project=<guid>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "project",
				Destination: &pc.project,
				Usage:       "the project guid ",
			},
		},
	}
}

func (pc *projectCmd) projectAction(ctx *cli.Context) error {
	var urlTemplate = "%s/box/api/projects/%s"
	uData, err := pc.store.ReadUserData()
	if err != nil {
		return err
	}
	if pc.project == "" && uData.ActiveProject != "" {
		pc.project = uData.ActiveProject
	}
	if pc.project == "" {
		return cli.NewExitError("expeced a project guid. Use --project", 1)
	}
	url := fmt.Sprintf(urlTemplate, uData.Host, pc.project)
	newrequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return cli.NewExitError("could not create new request object "+err.Error(), 1)
	}
	// create a cookie
	cookie := http.Cookie{Name: "feedhenry", Value: uData.Auth}
	newrequest.AddCookie(&cookie)
	res, err := pc.getter(newrequest)
	if err != nil {
		return cli.NewExitError("failed to make request to get project", 1)
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return cli.NewExitError("failed to read response body "+err.Error(), 1)
	}
	if res.StatusCode != http.StatusOK {
		pc.out.Write(data)
		return cli.NewExitError(fmt.Sprintf("\n unexpected response %d \n", res.StatusCode), 1)
	}
	projectModel := &commands.Project{}
	if err := json.Unmarshal(data, projectModel); err != nil {
		return cli.NewExitError("failed to Unmarshal json "+err.Error(), 1)
	}
	t := template.New("project")
	t, err = t.Parse(projectTemplate)
	if err != nil {
		return cli.NewExitError("failed to parse template "+err.Error(), 1)
	}
	//template, writer
	if err := t.Execute(pc.out, projectModel); err != nil {
		return cli.NewExitError("failed to output template "+err.Error(), 1)
	}
	return nil
}

// NewProjectCmd configures a new project command
func NewProjectCmd(in io.Reader, out io.Writer, store storage.Storer) cli.Command {
	pc := &projectCmd{
		in:     in,
		out:    out,
		store:  store,
		getter: http.DefaultClient.Do,
	}
	return pc.Project()
}
