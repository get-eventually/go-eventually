<!-- markdownlint-disable-file MD033 -->
<!-- markdownlint-disable-file MD041 -->

<br />
<div align="center">
    <img alt="Eventually" src = "./resources/logo.png" width = 300>
</div>
<br />
<div align="center">
    <strong>
        Domain-driven Design, Event Sourcing and CQRS for Go
    </strong>
</div>
<br />
<div align="center">
    <!-- Code Coverage -->
    <a href="https://app.codecov.io/gh/get-eventually/go-eventually">
        <img alt="Codecov" src="https://img.shields.io/codecov/c/github/get-eventually/go-eventually?style=flat-square">
    </a>
    <!-- pkg.go.dev -->
    <a href="https://pkg.go.dev/github.com/get-eventually/go-eventually">
        <img alt="Go Reference"
        src="https://pkg.go.dev/badge/github.com/get-eventually/go-eventually.svg">
    </a>
    <!-- License -->
    <a href="./LICENSE">
        <img alt="GitHub license"
        src="https://img.shields.io/github/license/get-eventually/go-eventually?style=flat-square">
    </a>
</div>
<br />

> [!WARNING]
> Though used in production environment, the library is still under active development.

<!-- markdownlint false positive -->

> [!NOTE]
> Prior to `v1` release the following Semantic Versioning
is being adopted:
>
> * Breaking changes are tagged with a new **minor** release,
> * New features, patches and documentation are tagged with a new **patch** release.

## Overview

`eventually` is a library providing abstractions and components to help you:

* Build Domain-driven Design-oriented code, (Domain Events, Aggregate Root, etc.)

* Reduce complexity of your code by providing ready-to-use components, such as PostgreSQL repository implementation, OpenTelemetry instrumentation, etc.

* Implement event-driven architectural patterns in your application, such as Event Sourcing or CQRS.

### How to install

You can add this library to your project by running:

```sh
go get -u github.com/get-eventually/go-eventually
```

## Contributing

Thank you for your consideration ❤️ You can head over our [CONTRIBUTING](./CONTRIBUTING.md) page to get started.

## License

This project is licensed under the [MIT license](LICENSE).

### Contribution

Unless you explicitly state otherwise, any contribution intentionally submitted for inclusion in `go-eventually` by you, shall be licensed as MIT, without any additional terms or conditions.
