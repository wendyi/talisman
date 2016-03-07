package git_repo

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"fmt"

	log "github.com/wendyi/logrus"
)

//FilePath represents the absolute path of an added file
type FilePath string

//FileName represents the base name of an added file
type FileName string

//Addition represents the end state of a file
type Addition struct {
	Path FilePath
	Name FileName
	Data []byte
}

//GitRepo represents a Git repository located at the absolute path represented by root
type GitRepo struct {
	root string
}

//RepoLocatedAt returns a new GitRepo with it's root located at the location specified by the argument.
//If the argument is not an absolute path, it will be turned into one.
func RepoLocatedAt(path string) GitRepo {
	fmt.Println(path)
	absoluteRoot, _ := filepath.Abs(path)
	return GitRepo{absoluteRoot}
}

//AllAdditions returns all the outgoing additions and modifications in a GitRepo. This does not include files that were deleted.
func (repo GitRepo) AllAdditions(path string) []Addition {
	return repo.Additions("origin/master", "master", path)
}

//Additions returns the outgoing additions and modifications in a GitRepo that are in the given commit range. This does not include files that were deleted.
func (repo GitRepo) Additions(oldCommit string, newCommit string, path string) []Addition {
	fmt.Println("Additions")
	files := repo.outgoingNonDeletedFiles(oldCommit, newCommit, path)
	result := make([]Addition, len(files))
	for i, file := range files {
		data, _ := repo.ReadRepoFile(file)
		result[i] = NewAddition(file, data)
	}
	log.WithFields(log.Fields{
		"oldCommit": oldCommit,
		"newCommit": newCommit,
		"additions": result,
	}).Info("Generating all additions in range.")
	fmt.Printf("%d", len(result))
	return result
}

//NewAddition returns a new Addition for a file with supplied name and contents
func NewAddition(filePath string, content []byte) Addition {
	return Addition{
		Path: FilePath(filePath),
		Name: FileName(path.Base(filePath)),
		Data: content,
	}
}

//ReadRepoFile returns the contents of the supplied relative filename by locating it in the git repo
func (repo GitRepo) ReadRepoFile(fileName string) ([]byte, error) {
	return ioutil.ReadFile(path.Join(repo.root, fileName))
}

//ReadRepoFileOrNothing returns the contents of the supplied relative filename by locating it in the git repo.
//If the given file cannot be located in theb repo, then an empty array of bytes is returned for the content.
func (repo GitRepo) ReadRepoFileOrNothing(fileName string) ([]byte, error) {
	filepath := path.Join(repo.root, fileName)
	if _, err := os.Stat(filepath); err == nil {
		return repo.ReadRepoFile(fileName)
	}
	return make([]byte, 0), nil
}

//Matches states whether the addition matches the given pattern.
//If the pattern ends in a path separator, then all files inside a directory with that name are matched. However, files with that name itself will not be matched.
//If a pattern contains the path separator in any other location, the match works according to the pattern logic of the default golang glob mechanism
//If there is no path separator anywhere in the pattern, the pattern is matched against the base name of the file. Thus, the pattern will match files with that name anywhere in the repository.
func (a Addition) Matches(pattern string) bool {
	var result bool
	if pattern[len(pattern)-1] == os.PathSeparator {
		result = strings.HasPrefix(string(a.Path), pattern)
	} else if strings.ContainsRune(pattern, os.PathSeparator) {
		result, _ = path.Match(pattern, string(a.Path))
	} else {
		result, _ = path.Match(pattern, string(a.Name))
	}
	log.WithFields(log.Fields{
		"pattern":  pattern,
		"filePath": a.Path,
		"match":    result,
	}).Debug("Checking addition for match.")
	return result
}

func (repo GitRepo) outgoingNonDeletedFiles(oldCommit, newCommit string, path string) []string {
	fmt.Println("outgoingNonDeletedFiles")
	allChanges := strings.Split(repo.fetchRawOutgoingDiff(oldCommit, newCommit, path), "\n")
	var result []string
	for _, c := range allChanges {
		if len(c) != 0 {
			result = append(result, c)
		}
	}
	return result
}

func (repo GitRepo) fetchRawOutgoingDiff(oldCommit string, newCommit string, path string) string {
	fmt.Println("fetchRawOutgoingDiff")
	gitRange := oldCommit + ".." + newCommit
	return string(repo.executeRepoCommand("git", "diff", gitRange, "--name-only", "--diff-filter=ACM", path))
}

func (repo GitRepo) executeRepoCommand(commandName string, args ...string) []byte {
	fmt.Println("executeRepoCommand")
	log.WithFields(log.Fields{
		"command": commandName,
		"args":    args,
	}).Debug("Building repo command")
	result := exec.Command(commandName, args...)
	result.Dir = repo.root
	o, err := result.Output()
	logEntry := log.WithFields(log.Fields{
		"output": string(o),
	})
	if err == nil {
		logEntry.Debug("Command excuted successfully")
	} else {
		logEntry.WithError(err).Fatal("Command execution failed")
	}
	return o
}
