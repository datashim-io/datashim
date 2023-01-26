# Git workflow for Datashim development

We'll roughly follow the Github development flow [used by the Kubernetes project](https://www.kubernetes.dev/docs/guide/github-workflow/). 

1. Visit https://github.com/datashim-io/datashim. Fork your own copy of Datashim to your Github account. For the sake of illustration, let's say this fork corresponds to `https://github.com/$user/datashim` where `$user` is your username.

2. Go to the source directory of your Go workspace and clone your fork there. Using the example above where the workspace is in `$HOME/goprojects`,
   ```
   $> mkdir -p $HOME/goprojects/src
   $> cd $HOME/goprojects/src
   $> git clone https://github.com/$user/datashim.git
   ```

3. Set the Datashim repo as your upstream and rebase
   ```
   $> cd $HOME/goprojects/src/datashim
   $> git remote add upstream https://github.com/datashim-io/datashim
   $> git remote set-url --push upstream no_push 
   ```
   The last line prevents pushing to upstream. You can verify your remotes by `git remote -v`
   ```
   $> git fetch upstream
   $> git checkout master
   $> git rebase upstream/master
   ```

4. Create a new branch to work on a feature or fix. Before this, please create an issue in the main Datashim repository that describes the problem or feature. Note the issue number (e.g. `nnn`) and assign it to yourself. In your local repository, create a branch to work on the fix. Use a short title (2 or 3 words) formed from the issue title/description along with the issue number as the branch name 

   ```
   $> git checkout -b nnn-short-title
   ```
   Make your changes. Then commit your changes. [Always sign your commits](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits)
   ```
   $> git commit -s -m "short descriptive message"
   $> git push $your_remote nnn-short-title
   ```

5. When you are ready to submit a Pull Request (PR) for your completed feature or branch, visit your fork on Github and click the button titled `Compare and Pull Request` next to your `nnn-short-title` branch. This will submit the PR to Datashim.io for review
   
6. After the review, prepare your PR for merging by [squashing your commits](https://medium.com/@slamflipstrom/a-beginners-guide-to-squashing-commits-with-git-rebase-8185cf6e62ec). 


