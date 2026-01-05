package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	count := flag.Int("count", 10000, "number of items to generate")
	dataType := flag.String("type", "paths", "type of data (paths, commands, mixed)")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed for reproducibility")
	flag.Parse()

	rand.Seed(*seed)

	switch *dataType {
	case "paths":
		generatePaths(*count)
	case "commands":
		generateCommands(*count)
	case "mixed":
		generateMixed(*count)
	default:
		fmt.Fprintf(os.Stderr, "Unknown type: %s\n", *dataType)
		os.Exit(1)
	}
}

// generatePaths generates realistic file paths
func generatePaths(count int) {
	prefixes := []string{
		"src", "pkg", "internal", "cmd", "test", "docs", "scripts",
		"config", "api", "lib", "vendor", "build", "tools",
	}

	components := []string{
		"service", "handler", "controller", "model", "view", "util",
		"helper", "manager", "processor", "validator", "converter",
		"repository", "entity", "dto", "middleware", "interceptor",
	}

	suffixes := []string{
		".go", ".ts", ".js", ".py", ".java", ".rs", ".c", ".cpp",
		".h", ".hpp", ".rb", ".php", ".sh", ".yaml", ".json", ".md",
	}

	for i := 0; i < count; i++ {
		depth := rand.Intn(5) + 1
		parts := make([]string, depth+1)

		parts[0] = prefixes[rand.Intn(len(prefixes))]

		for j := 1; j < depth; j++ {
			parts[j] = components[rand.Intn(len(components))]
		}

		filename := fmt.Sprintf("%s_%d%s",
			components[rand.Intn(len(components))],
			rand.Intn(1000),
			suffixes[rand.Intn(len(suffixes))])
		parts[depth] = filename

		fmt.Println(strings.Join(parts, "/"))
	}
}

// generateCommands generates realistic command-line commands
func generateCommands(count int) {
	commands := []string{
		"git commit -m",
		"docker build -t",
		"kubectl apply -f",
		"npm install",
		"go build",
		"cargo run",
		"python -m",
		"make",
		"terraform apply",
		"ansible-playbook",
		"systemctl restart",
		"journalctl -u",
		"curl -X POST",
		"wget",
		"ssh",
		"rsync -av",
		"tar -xzvf",
		"grep -r",
		"find . -name",
		"ps aux | grep",
	}

	args := []string{
		"production", "staging", "development", "test",
		"service", "application", "database", "cache",
		"frontend", "backend", "api", "worker",
		"config", "deployment", "migration", "backup",
	}

	for i := 0; i < count; i++ {
		cmd := commands[rand.Intn(len(commands))]
		argCount := rand.Intn(3) + 1
		cmdArgs := make([]string, argCount)

		for j := 0; j < argCount; j++ {
			cmdArgs[j] = args[rand.Intn(len(args))]
		}

		fmt.Printf("%s %s_%d\n", cmd, strings.Join(cmdArgs, "_"), i)
	}
}

// generateMixed generates a mix of different types
func generateMixed(count int) {
	// 50% paths, 30% commands, 20% other
	pathCount := count / 2
	cmdCount := count * 3 / 10

	// Interleave them randomly
	types := make([]string, count)
	for i := 0; i < pathCount; i++ {
		types[i] = "path"
	}
	for i := pathCount; i < pathCount+cmdCount; i++ {
		types[i] = "cmd"
	}
	for i := pathCount + cmdCount; i < count; i++ {
		types[i] = "other"
	}

	// Shuffle
	rand.Shuffle(len(types), func(i, j int) {
		types[i], types[j] = types[j], types[i]
	})

	// Generate based on shuffled types
	for _, t := range types {
		switch t {
		case "path":
			generatePaths(1)
		case "cmd":
			generateCommands(1)
		case "other":
			generateOther(1)
		}
	}
}

// generateOther generates other types of data
func generateOther(count int) {
	categories := []string{
		"User: %s <%s@example.com>",
		"Issue #%d: %s",
		"PR #%d: %s",
		"Branch: feature/%s-%d",
		"Tag: v%d.%d.%d",
		"Commit: %s",
		"Server: %s-%d.prod.example.com",
	}

	names := []string{
		"john", "jane", "alice", "bob", "charlie", "diana",
		"fix", "feat", "refactor", "update", "improve", "add",
	}

	for i := 0; i < count; i++ {
		category := categories[rand.Intn(len(categories))]
		name := names[rand.Intn(len(names))]

		switch category {
		case "User: %s <%s@example.com>":
			fmt.Printf(category, name, name)
		case "Issue #%d: %s", "PR #%d: %s":
			fmt.Printf(category, rand.Intn(10000), name)
		case "Branch: feature/%s-%d":
			fmt.Printf(category, name, rand.Intn(1000))
		case "Tag: v%d.%d.%d":
			fmt.Printf(category, rand.Intn(5), rand.Intn(20), rand.Intn(100))
		case "Commit: %s":
			// Generate random hex string
			fmt.Printf(category, fmt.Sprintf("%x", rand.Int63()))
		case "Server: %s-%d.prod.example.com":
			fmt.Printf(category, name, rand.Intn(100))
		}
		fmt.Println()
	}
}
