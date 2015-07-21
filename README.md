# tracksuit

keeps Github issues and Tracker stories in sync by doing the following:

* if an issue exists with no stories labelled with `reponame/username#123`,
  create a story with the issue's name/description

* for every issue, find all labelled stories and reflect their status in a
  helpful comment (only one comment per issue, updating if it already exists)

* if an issue is open and all stories for it are accepted, close it with a
  message linking to the stories, and instructing the user to reopen if they
  have any questions or more feedback

it is implemented as a stateless CLI, and you just give it flags for the Github
organization and Tracker project ID to sync up. comments and stories will be
created as the user for the respective tokens.
