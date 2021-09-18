// TTW Software Team
// Mathis Van Eetvelde
// 2021-present

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
    "strconv"
	"strings"

	"github.com/sethvargo/go-githubactions"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	scope = "https://www.googleapis.com/auth/drive.file"
	filenameInput = "filename"
	nameInput = "name"
	folderIdInput = "folderId"
	credentialsInput = "credentials"
    updateInput = "update"
)

func uploadNewFileToDrive(svc *drive.Service, filename string, folderId string, name string) {
    file, err := os.Open(filename)
    if err != nil {
        githubactions.Fatalf(fmt.Sprintf("opening file with filename: %v failed with error: %v", filename, err))
    }

    f := &drive.File {
        Name: name,
        Parents: []string{folderId},
    }
    _, err = svc.Files.Create(f).Media(file).Do()

    if err != nil {
        githubactions.Fatalf(fmt.Sprintf("Uploading new file failed with error: %v", err))
    } else {
        githubactions.Debugf("Uploaded new file successfully.")
    }
}

func updateFileOnDrive(svc *drive.Service, filename string, folderId string, driveFile *drive.File, name string) {
    file, err := os.Open(filename)
    if err != nil {
        githubactions.Fatalf(fmt.Sprintf("opening file with filename: %v failed with error: %v", filename, err))
    }

    f := &drive.File {
        Name: name,
    }
    _, err = svc.Files.Update(driveFile.Id, f).Media(file).Do()

    if err != nil {
        githubactions.Fatalf(fmt.Sprintf("Updating file failed with error: %v", err))
    } else {
        githubactions.Debugf("Updated file successfully.")
    }
}

func main() {

	// get filename argument from action input
	filename := githubactions.GetInput(filenameInput)
	if filename == "" {
		missingInput(filenameInput)
	}

	// get name argument from action input
	name := githubactions.GetInput(nameInput)

	// get folderId argument from action input
	folderId := githubactions.GetInput(folderIdInput)
	if folderId == "" {
		missingInput(folderIdInput)
	}

	// get base64 encoded credentials argument from action input
	credentials := githubactions.GetInput(credentialsInput)
	if credentials == "" {
		missingInput(credentialsInput)
	}

    // get update flag
    var updateFlag bool
    update := githubactions.GetInput(updateInput)
    if update == "" {
        githubactions.Warningf("Update is disabled.")
        updateFlag = false
    } else {
        updateFlag, _ = strconv.ParseBool(update)
    }

	// add base64 encoded credentials argument to mask
	githubactions.AddMask(credentials)

	// decode credentials to []byte
	decodedCredentials, err := base64.StdEncoding.DecodeString(credentials)
	if err != nil {
		githubactions.Fatalf(fmt.Sprintf("base64 decoding of 'credentials' failed with error: %v", err))
	}

	creds := strings.TrimSuffix(string(decodedCredentials), "\n")

	// add decoded credentials argument to mask
	githubactions.AddMask(creds)

	// fetching a JWT config with credentials and the right scope
	conf, err := google.JWTConfigFromJSON([]byte(creds), scope)
	if err != nil {
		githubactions.Fatalf(fmt.Sprintf("fetching JWT credentials failed with error: %v", err))
	}

	// instantiating a new drive service
	ctx := context.Background()
	svc, err := drive.New(conf.Client(ctx))
	if err != nil {
		log.Println(err)
	}

	file, err := os.Open(filename)
	if err != nil {
		githubactions.Fatalf(fmt.Sprintf("opening file with filename: %v failed with error: %v", filename, err))
	}

	// decide name of file in GDrive
	if name == "" {
		name = file.Name()
	}

    if updateFlag {
        fmt.Println("Updating file on drive: $s", name)
        // Query for all files in google drive directory with name = <name>
        var nameQuery string
        nameQuery = fmt.Sprintf("name = '%s' and trashed = false", name)
        filesQueryCallResult, err := svc.Files.List().Q(nameQuery).Do()

        if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
			fmt.Println("Unable to retrieve files")
		}

        if len(filesQueryCallResult.Files) == 0 {
            // Upload new file to google drive
            uploadNewFileToDrive(svc, filename, folderId, name)
        } else {
            // Update file on google drive
            for _, driveFile := range filesQueryCallResult.Files {
                fmt.Printf("Updating file; %s (%s) Trashed=$t\n", driveFile.Name, driveFile.Id, driveFile.Trashed)
                updateFileOnDrive(svc, filename, folderId, driveFile, name)
            }
        }
    } else {
        fmt.Println("Uploading new file on drive: $s", name)
        uploadNewFileToDrive(svc, filename, folderId, name)
    }

}

func missingInput(inputName string) {
	githubactions.Fatalf(fmt.Sprintf("missing input '%v'", inputName))
}
