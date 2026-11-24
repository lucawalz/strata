# Changelog

## [0.1.4](https://github.com/lucawalz/strata/compare/v0.1.3...v0.1.4) (2026-11-24)


### Bug Fixes

* **skills:** run the repository check recipe before opening a pull request ([31203a3](https://github.com/lucawalz/strata/commit/31203a376efaefee394ee79cf5a6616133663db2))

## [0.1.3](https://github.com/lucawalz/strata/compare/v0.1.2...v0.1.3) (2026-11-23)


### Bug Fixes

* **check:** scan only files git tracks or does not ignore ([829821b](https://github.com/lucawalz/strata/commit/829821b32d5aceab84a1bb614811715ff72b2651))
* **check:** skip nested repositories and directories in the tree check ([374acd0](https://github.com/lucawalz/strata/commit/374acd0a8e1f9ff1b46963456f6b089234ff444d))
* **skills:** stop capture refusals from quoting the found identifier ([585d36f](https://github.com/lucawalz/strata/commit/585d36fc223f50d73101b47e27c7bb801129a65a))
* **workflows:** authenticate the gate app with its client id ([e97c811](https://github.com/lucawalz/strata/commit/e97c81179e4d6e8f6cd56aac511900e04eef1e89))

## [0.1.2](https://github.com/lucawalz/strata/compare/v0.1.1...v0.1.2) (2026-11-23)


### Bug Fixes

* **skills:** keep refs for outside records and link notes in the body ([adf8985](https://github.com/lucawalz/strata/commit/adf898558b400fd73b42b7c9a593482c348f91fe))
* **workflows:** scan for secrets with a config the pull request cannot change ([6fe5061](https://github.com/lucawalz/strata/commit/6fe506197eefbedb9ceb6896ce9bdf588cde6517))

## [0.1.1](https://github.com/lucawalz/strata/compare/v0.1.0...v0.1.1) (2026-11-23)


### Bug Fixes

* **skills:** keep personal identifiers out of the shipped skill scenarios ([090e2f0](https://github.com/lucawalz/strata/commit/090e2f00a0cdd95f35be2b44454f8e28b4a63e98))

## 0.1.0 (2026-11-23)


### Features

* **calibrate:** measure characters per token with the token counting api ([5956906](https://github.com/lucawalz/strata/commit/59569069a2b717b88271a087e739c85559dad088))
* **check:** add governance signature isolation and raw immutability checks ([134756e](https://github.com/lucawalz/strata/commit/134756e6921a1c50c423670a24eced3ee75a2310))
* **check:** add tree checks for schema size structure and links ([c30453f](https://github.com/lucawalz/strata/commit/c30453f8fac86595d5d9e21ebcb2da79e7824bf6))
* **check:** audit governance commits that reach the main branch ([363cdf4](https://github.com/lucawalz/strata/commit/363cdf4353aade7c925857c021e01205913b0443))
* **cli:** wire check and audit commands with github annotations ([1392874](https://github.com/lucawalz/strata/commit/1392874682eedb4708bdcbfe662f73964a83e886))
* **gitx:** add git helpers for ranges signatures and diffs ([e931c1b](https://github.com/lucawalz/strata/commit/e931c1bbb1a84174d64599e10c3ef9b1be63f4bc))
* **inject:** detect hidden unicode html and exfiltration patterns in notes ([592edc7](https://github.com/lucawalz/strata/commit/592edc73579283af2e8386919dade74e4ea673f3))
* **note:** parse frontmatter headings and links from markdown notes ([354ef0e](https://github.com/lucawalz/strata/commit/354ef0e157ac10c5e06c0fd3a754deebb76ca962))
* **pii:** validate iban card tax and identity numbers with checksums ([f57f516](https://github.com/lucawalz/strata/commit/f57f516107c185d98b1867ecb470b127ee56cc82))
* **schema:** add note and configuration schemas with defaults ([0049fb8](https://github.com/lucawalz/strata/commit/0049fb83a66101c7ab51ca8f309657c72a247862))
* **skills:** add capture and curate skills with a sync command ([5b6213a](https://github.com/lucawalz/strata/commit/5b6213a96693100c769625c35fd2d9afafab63f5))
* **workflows:** add a gate workflow that reports through a dedicated app ([3bbc80a](https://github.com/lucawalz/strata/commit/3bbc80a14d8bba1d9df78266e957291a5d96cf3d))
* **workflows:** add reusable check and audit workflows for notes repos ([6a45715](https://github.com/lucawalz/strata/commit/6a4571517c99b0deaabed6a732711f6eb2b64c18))


### Bug Fixes

* **calibrate:** count notes without frontmatter and name failing files ([329bedf](https://github.com/lucawalz/strata/commit/329bedfe30e7fca7165b9b4e3ec33dc95bd88ae2))
* **calibrate:** skip empty notes instead of sending them to count_tokens ([29480d3](https://github.com/lucawalz/strata/commit/29480d3af476d77f246d72e73a36cc2ec6c3969c))
* **check:** close governance bypasses through renames merges and path variants ([29eace1](https://github.com/lucawalz/strata/commit/29eace1eeb4ff69bcaf67da9137e7859fc29e547))
* **check:** fail squashed governance commits and judge merges by their own changes ([81f8fed](https://github.com/lucawalz/strata/commit/81f8fedf0d5b0e820666895d3c97f3468d00cfcb))
* **check:** govern agent files at any depth and the signer config ([c30c4bb](https://github.com/lucawalz/strata/commit/c30c4bbba4318f03859340b328d28568c0cc013a))
* **check:** judge append-only edits against the merge base of the range ([1caa7c7](https://github.com/lucawalz/strata/commit/1caa7c7e8a86bdd6de317436fe36de409ff5fe93))
* **check:** limit size and heading checks to notes outside the archive ([b06016f](https://github.com/lucawalz/strata/commit/b06016f2273a9a1fc4573ee75f6056049a743a6c))
* **check:** read audit configuration and signers from the previous tip ([efe5a2f](https://github.com/lucawalz/strata/commit/efe5a2f593ddc4f3c8954fc3a39dd0f3bd1b24d2))
* **check:** treat renames into raw as additions and flag renames out ([ed3dee1](https://github.com/lucawalz/strata/commit/ed3dee1bc0c6a496120f7765a27994d69d2ad982))
* **ci:** pin golangci-lint 2.15.0 built with go 1.27 ([2e087b9](https://github.com/lucawalz/strata/commit/2e087b9e877e29dc4e5884d1df0413bd3aebcbcf))
* **note:** report file line numbers for links headings and yaml errors ([e6e009a](https://github.com/lucawalz/strata/commit/e6e009a31b65e42094e788c786dd7539815879ba))
* **pii:** cut false positives for year lists phones serials and emoji ([6f80da9](https://github.com/lucawalz/strata/commit/6f80da906d291c09120c22d345cb668d29869306))
* **pii:** find numbers next to other digit groups and sort matches ([a9f857c](https://github.com/lucawalz/strata/commit/a9f857cfaa8a6faeca8bf6cd4d1fc18f6d405b02))
* **pii:** normalise separators and case before validating printed numbers ([ba92523](https://github.com/lucawalz/strata/commit/ba9252348628c547c479938e083f7ba728c6f539))
* **skills:** give synced skill directories the regular directory mode ([8c79c09](https://github.com/lucawalz/strata/commit/8c79c0965c9c4dcca55a0e72611e13e12a01b1ff))
* **skills:** keep curated inbox items in raw and replace skills atomically ([041188d](https://github.com/lucawalz/strata/commit/041188d4af29543ea8b73b8260c5e14143dfc693))
* **workflows:** build the checker from a fetch of the workflow commit ([c747897](https://github.com/lucawalz/strata/commit/c7478975239a3707b556dcdfe1dfa22cd85ee72f))
* **workflows:** check out the pr head and always run the diff ([13a0ae0](https://github.com/lucawalz/strata/commit/13a0ae07291db0b997cb48a3f59949d342a4c7f2))
* **workflows:** file audit issues only for unsigned commits and validate inputs ([f43afe5](https://github.com/lucawalz/strata/commit/f43afe51f275c09a2fe0de02b9002d4bfdc38a44))
* **workflows:** keep gate secrets on the environment and document the caller ([a9e8872](https://github.com/lucawalz/strata/commit/a9e88725a960fe80ea6f2419594cffc847c84d3e))
* **workflows:** list audit findings in the issue and title config failures ([c58eb53](https://github.com/lucawalz/strata/commit/c58eb53fbb8cf8998e923f7c97147bc4ffaf56ed))
* **workflows:** run the installed strata binary so audit exit codes survive ([fdbaa45](https://github.com/lucawalz/strata/commit/fdbaa459fd58599e5865a86c4dee1ab00354aac5))
