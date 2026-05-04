# Changelog

## [0.4.2](https://github.com/get-eventually/go-eventually/compare/v0.4.1...v0.4.2) (2026-05-04)


### Bug Fixes

* **deps:** update google.golang.org/genproto digest to 7cedc36 ([#320](https://github.com/get-eventually/go-eventually/issues/320)) ([f838c21](https://github.com/get-eventually/go-eventually/commit/f838c21529199f80cd8668694d9ee8bc90b6033a))

## [0.4.1](https://github.com/get-eventually/go-eventually/compare/v0.4.0...v0.4.1) (2026-04-22)

> **Note:** This release retracts `v0.4.0`, which was published prematurely without the `message.Stream` iterator migration. Users on v0.4.0 should upgrade to v0.4.1.

### ⚠ BREAKING CHANGES

* replace channel-based event streaming with message.Stream iterator ([#330](https://github.com/get-eventually/go-eventually/issues/330))

### Features

* **examples:** add todolist connect example exercising the new streaming API ([#331](https://github.com/get-eventually/go-eventually/issues/331)) ([492525f](https://github.com/get-eventually/go-eventually/commit/492525f904d6dc1318460c008fb100e08350ebba))
* replace channel-based event streaming with message.Stream iterator ([#330](https://github.com/get-eventually/go-eventually/issues/330)) ([a96b37a](https://github.com/get-eventually/go-eventually/commit/a96b37a9f3f683e200670c76ffc7d4338901c885))


### Bug Fixes

* **release-please:** enable bump-*-pre-major feature flags ([f8456ab](https://github.com/get-eventually/go-eventually/commit/f8456ab24d7816e29b90fa78fc76224160f51747))
* **release:** drop package-name from release-please config ([#328](https://github.com/get-eventually/go-eventually/issues/328)) ([6ae1f8e](https://github.com/get-eventually/go-eventually/commit/6ae1f8e0bc9163ff708b43df1429a1d51d2afaf1))
* rephrase docs ([1aa4425](https://github.com/get-eventually/go-eventually/commit/1aa4425ea048def137b7e62d26358b302ea1d866))


### Documentation

* **README:** add the How to Use section ([baddbb7](https://github.com/get-eventually/go-eventually/commit/baddbb7ce0e7c54c0299071793d2720e2e34405a))


### Miscellaneous

* release 0.4.1 ([f34e71c](https://github.com/get-eventually/go-eventually/commit/f34e71c6caaf4ad86537728dcb14df4459e7193c))

## [0.4.0](https://github.com/get-eventually/go-eventually/compare/v0.3.0...v0.4.0) (2026-04-21)


### Features

* drop dependabot, move to renovate ([fb6883f](https://github.com/get-eventually/go-eventually/commit/fb6883ff3113a7c79bbf6325c83ed829af5fc461))
* **renovate:** enable automerging ([6b4a647](https://github.com/get-eventually/go-eventually/commit/6b4a647997f0aee8c70bb63ce8153bd38bccddbe))
* use Nix flake for lint and test flows ([72ad95f](https://github.com/get-eventually/go-eventually/commit/72ad95fca3f89efaf52b2e18c9cd1231b3dd8d66))


### Bug Fixes

* **deps:** update all non-major dependencies ([#280](https://github.com/get-eventually/go-eventually/issues/280)) ([703da00](https://github.com/get-eventually/go-eventually/commit/703da000639b92f5fbcb67e061f53566a2b95417))
* **deps:** update google.golang.org/genproto digest to 3122310 ([#299](https://github.com/get-eventually/go-eventually/issues/299)) ([82b0730](https://github.com/get-eventually/go-eventually/commit/82b07305c54ff4c3a71cac3dc58329c248b29b47))
* **deps:** update google.golang.org/genproto digest to 9702482 ([#303](https://github.com/get-eventually/go-eventually/issues/303)) ([e73acf0](https://github.com/get-eventually/go-eventually/commit/e73acf03f24ea944cde658b1323be872ac01074d))
* **deps:** update google.golang.org/genproto digest to a7a43d2 ([#283](https://github.com/get-eventually/go-eventually/issues/283)) ([4039e97](https://github.com/get-eventually/go-eventually/commit/4039e977a05432565193c74e856a54caafea508e))
* **deps:** update google.golang.org/genproto digest to ee84b53 ([#277](https://github.com/get-eventually/go-eventually/issues/277)) ([f984d9c](https://github.com/get-eventually/go-eventually/commit/f984d9ca2de3b9c4ad84ddf9454b590576c84afd))
* **deps:** update module google.golang.org/grpc to v1.79.3 [security] ([#317](https://github.com/get-eventually/go-eventually/issues/317)) ([5b284e2](https://github.com/get-eventually/go-eventually/commit/5b284e2b7d6b5926dbdf894a216f36a62ec02ab9))
* pin commit sha for release-please bootstrap ([8447592](https://github.com/get-eventually/go-eventually/commit/8447592d859af6f53415fdb5d6d1b5a5597cd97b))
