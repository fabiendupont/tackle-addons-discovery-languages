module github.com/konveyor/tackle-addons-discovery-languages

go 1.16

require (
	github.com/go-git/go-git/v5 v5.4.2
	github.com/konveyor/tackle-hub v0.0.0-00010101000000-000000000000
)

replace github.com/konveyor/tackle-hub => github.com/mansam/tackle-hub v0.0.0-20211209180206-49c15bbac0b5

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20181213151034-8d9ed539ba31

replace k8s.io/api => k8s.io/api v0.0.0-20181213150558-05914d821849

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20181213153335-0fe22c71c476
