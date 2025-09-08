Gem::Specification.new do |spec|
  spec.name          = "store-service"
  spec.version       = "0.1.0"
  spec.authors       = ["RinseCRM Team"]
  spec.email         = ["team@rinsecrm.com"]

  spec.summary       = "Ruby client for Store Service gRPC service"
  spec.description   = "A Ruby gem providing protobuf definitions and gRPC client for the Store Service"
  spec.homepage      = "https://github.com/rinsecrm/store-service"
  spec.license       = "MIT"
  spec.required_ruby_version = Gem::Requirement.new(">= 2.7.0")

  spec.metadata["homepage_uri"] = spec.homepage
  spec.metadata["source_code_uri"] = spec.homepage
  spec.metadata["changelog_uri"] = "#{spec.homepage}/blob/main/CHANGELOG.md"

  # Specify which files should be added to the gem when it is released.
  spec.files = Dir.glob("lib/**/*") + Dir.glob("*.proto") + %w[README.md LICENSE gemspec]
  spec.require_paths = ["lib"]

  spec.add_dependency "grpc", "~> 1.54"
  spec.add_dependency "google-protobuf", "~> 3.21"

  spec.add_development_dependency "bundler", "~> 2.0"
  spec.add_development_dependency "rake", "~> 13.0"
end
