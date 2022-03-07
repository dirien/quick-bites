# GoReleaser: How To Distribute Your Binaries With Homebrew or GoFish
This article is going to be a quick bite (or drink)! We going to discover, how fast we can create a Homebrew or GoFish deployment of our binaries with the help of GoReleaser.
But first, let us take a look into the concepts of the two package managers:

## Homebrew 🍺
The Missing Package Manager for macOS (or Linux)
This statement is not from me, but from the official Homebrew website. Homebrew is similar to other packages managers, apt-get, aptitude, or dpkg. I will not go in this article into the details of Homebrew but some terms are important to understand, as we going to use them in our .gorleaser.yaml file:
Tap: A Git repository of packages.
Formula: A software package. When we want to install new programs or libraries, we install a formula.

## GoFish 🐠
GoFish, the Package Manager 🐠
GoFish is a cross-platform systems package manager, bringing the ease of use of Homebrew to Linux and Windows.
Same as we Homebrew, I am not going into detail of GoFish but we need also here some understanding of the GoFish terminology:
Rig: A git repository containing fish food.
Food: The package definition

## The example code
For each package manager, you should create its own GitHub repository. You can name it as you please, but i prefer to add the meaning of the repository.
- **goreleaser-rig** for GoFish
- **goreleaser-tap** for Homebrew

Add following snippet for GoFish support, to your existing **.goreleaser.yaml**:
```yaml
...
rigs:
- rig:
  owner: dirien
  name: goreleaser-rig
  homepage: "https://github.com/dirien/quick-bites"
  description: "Different type of projects, not big enough to warrant a separate repo."
  license: "Apache License 2.0"
...
```  
And for Homebrew, add this little snippet:
```yaml
...
brews:
- tap:
  owner: dirien
  name: goreleaser-tap
  folder: Formula
  homepage: "https://github.com/dirien/quick-bites"
  description: "Different type of projects, not big enough to warrant a separate repo."
  license: "Apache License 2.0"
  ...
```

That's all. You can now run the release process and will see this in your logs:

```shell
• homebrew tap formula
 • pushing                   formula=Formula/goreleaser-brew-fish.rb repo=dirien/goreleaser-tap
• gofish fish food cookbook
 • pushing                   food=Food/goreleaser-brew-fish.lua repo=dirien/goreleaser-rig
```


# The End
Now you can distribute this tap or rig repositories and everybody can install your projects via this package manager.