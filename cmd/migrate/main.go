package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tig-company/featury/internal/database"
)

var (
	configFile      string
	migrationsDir   string
	databaseConfig  *database.Config
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration tool for Featury",
	Long:  `A CLI tool for managing database migrations in the Featury feature flag system.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
		setupDatabaseConfig()
	},
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	Long:  `Runs all pending migrations to bring the database up to the latest version.`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.Connect(databaseConfig)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		migrator := database.NewMigrator(db, migrationsDir)
		if err := migrator.Up(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback one migration",
	Long:  `Rolls back the most recent migration.`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.Connect(databaseConfig)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		migrator := database.NewMigrator(db, migrationsDir)
		if err := migrator.Down(); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Rollback all migrations",
	Long:  `Rolls back all migrations, effectively resetting the database to its initial state.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Are you sure you want to reset all migrations? This will delete all data! (y/N): ")
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return
		}

		db, err := database.Connect(databaseConfig)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		migrator := database.NewMigrator(db, migrationsDir)
		if err := migrator.Reset(); err != nil {
			log.Fatalf("Failed to reset migrations: %v", err)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current migration version",
	Long:  `Displays the current migration version and whether the database is in a dirty state.`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.Connect(databaseConfig)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		migrator := database.NewMigrator(db, migrationsDir)
		version, dirty, err := migrator.Version()
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
		}

		status := "clean"
		if dirty {
			status = "dirty"
		}

		fmt.Printf("Current migration version: %d (%s)\n", version, status)
	},
}

var forceCmd = &cobra.Command{
	Use:   "force [version]",
	Short: "Force set migration version",
	Long:  `Forces the migration version to a specific value without running migrations. Use with caution!`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}

		fmt.Printf("Are you sure you want to force migration version to %d? (y/N): ", version)
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return
		}

		db, err := database.Connect(databaseConfig)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		migrator := database.NewMigrator(db, migrationsDir)
		if err := migrator.Force(version); err != nil {
			log.Fatalf("Failed to force migration version: %v", err)
		}
	},
}

var dropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop all database objects",
	Long:  `Drops all database objects (tables, indexes, etc.). This is irreversible!`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Are you sure you want to drop all database objects? This will delete all data! (y/N): ")
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return
		}

		db, err := database.Connect(databaseConfig)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		migrator := database.NewMigrator(db, migrationsDir)
		if err := migrator.Drop(); err != nil {
			log.Fatalf("Failed to drop database objects: %v", err)
		}
	},
}

var createDbCmd = &cobra.Command{
	Use:   "create-db",
	Short: "Create the database if it doesn't exist",
	Long:  `Creates the database specified in the configuration if it doesn't already exist.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := database.CreateDatabase(databaseConfig); err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
	},
}

var dropDbCmd = &cobra.Command{
	Use:   "drop-db",
	Short: "Drop the database",
	Long:  `Drops the database specified in the configuration. This is irreversible!`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Are you sure you want to drop database '%s'? This will delete all data! (y/N): ", databaseConfig.Database)
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return
		}

		if err := database.DropDatabase(databaseConfig); err != nil {
			log.Fatalf("Failed to drop database: %v", err)
		}
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&migrationsDir, "migrations-dir", "", "path to migrations directory")
	
	// Add all commands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(forceCmd)
	rootCmd.AddCommand(dropCmd)
	rootCmd.AddCommand(createDbCmd)
	rootCmd.AddCommand(dropDbCmd)
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Look for config in the current directory
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Environment variables
	viper.SetEnvPrefix("FEATURY")
	viper.AutomaticEnv()

	// Default migrations directory
	if migrationsDir == "" {
		// Try to find migrations directory relative to the current executable
		if execDir, err := os.Executable(); err == nil {
			migrationsDir = filepath.Join(filepath.Dir(execDir), "..", "..", "internal", "database", "migrations")
		} else {
			migrationsDir = "internal/database/migrations"
		}
	}

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}
}

func setupDatabaseConfig() {
	databaseConfig = &database.Config{
		Host:     viper.GetString("database.host"),
		Port:     viper.GetInt("database.port"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		Database: viper.GetString("database.database"),
		SSLMode:  viper.GetString("database.sslmode"),
	}

	// Set defaults if not provided
	if databaseConfig.Host == "" {
		databaseConfig.Host = "localhost"
	}
	if databaseConfig.Port == 0 {
		databaseConfig.Port = 5432
	}
	if databaseConfig.User == "" {
		databaseConfig.User = "featury"
	}
	if databaseConfig.Password == "" {
		databaseConfig.Password = "featury"
	}
	if databaseConfig.Database == "" {
		databaseConfig.Database = "featury"
	}
	if databaseConfig.SSLMode == "" {
		databaseConfig.SSLMode = "disable"
	}
}