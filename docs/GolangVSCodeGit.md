# Recommended environment setup for development

## Setting up Go and VSCode

1. Visit https://go.dev/doc/install to download and install Go on your computer. Alternatively, you can also use package managers for your operating system (e..g Homebrew for macOS)

2. Once installed, run `go version` to verify that the installation is working

3. (Recommended) Go uses a variable `GOPATH` to [point to the current workspace](https://github.com/golang/go/wiki/SettingGOPATH). Package install commands such as `go install` will use this as their destination. If you are using a package as well as extending it, then it would be better to set up a separate workspace for development. To do this, create a separate directory, e.g. `$HOME/goprojects` and set it up with `bin`,`src`, and `pkg` sub-directories, and set `GOPATH` to point to it when developing. You can also use VSCode to modify `GOPATH` per project (see below)

4. Download VSCode from https://code.visualstudio.com/download. Open Extensions tab and search for Go or go to https://marketplace.visualstudio.com/items?itemName=golang.go. Verify that the extension is by Go team at Google. Install extension to VSCode and test it [with a sample program](https://docs.microsoft.com/en-us/azure/developer/go/configure-visual-studio-code)

### Setting up Datashim in VSCode

Before following the below suggestions, please ensure that you have checked out Datashim following the [git workflow for development](GitWorkflow.md). Datashim is a collection of multiple Go projects including the Dataset Operator, CSI-S3, Ceph Cache plugin, etc. Therefore, the VSCode setup is not as straightforward as with a single Go project. 

1. Start VSCode. Open a new window (**File** -> **New Window**). Select the Explorer view (generally the topmost icon on the left pane)

2. Add a folder to the workspace (**File** -> **Add Folder To Workspace**). In the file picker dialog, traverse to `$HOME/goprojects/src/github.com/$user/datashim` and then deeper into subprojects (i.e. `src/` folder). At this point, add the subfolder representing the project that you want to work on (e.g. `dataset-operator`). **Do not add the project root folder to the VSCode workspace**.

3. Your Explorer view will have the project in the side panel like so:
   
   ![](pictures/vscode-ws.png)

4. If you have followed the advice of having a separate directory for go projects, you need to inform Go plugin in VSCode about it. Open **Preferences** -> **Settings**. Click on **User** or **Workspace** tab. On the left pane, click on **Extensions** -> **Go** and scroll down to **Gopath** on the right-hand pane like so:

   ![](pictures/vscode-gopath.png) 

5. Add these lines to the JSON file:
   ```
   "go.toolsGopath": "$HOME/go",
   "go.gopath": "$HOME/goprojects",
   ```
   where the first line is the Go installation folder and the second line is the folder you've created for hacking.
