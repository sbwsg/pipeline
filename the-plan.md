Instead of doing the creds-init dance, let's instead do the following:
1. V Mount a unique /tekton/creds emptyDir for every Step. (forces 0777, world-writeable and empty, for each Step)
2. V ALWAYS set $(credentials.path) to /tekton/creds
3. V Get the secret volumes that creds-init used to process.
4. V Mount those secret volumes in every Step.
5. V Pass the flags that used to go to creds-init into every Step for their entrypoint.
6. V Entrypoint then responsible for copying creds out of secret volumes and into /tekton/creds.
7. V Entrypoint then copies creds out of /tekton/creds to $HOME.

Notes
- clear; egrep "creds-?init" -ir --exclude-dir .git --exclude-dir vendor --exclude the-plan.md .
- Move all remaining creds-init code into its own package for easy removal?


Things That Are Bad About Creds-Init
- Secrets are mounted as 0777 with root ownership in /tekton/creds-secrets
