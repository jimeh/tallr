package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	err := mainE()
	if err != nil {
		log.Fatal(err)
	}
}

func mainE() error {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT)

	rootCmd := &cobra.Command{
		Use: "tallr",
	}

	debugCmd := &cobra.Command{
		Use: "debug",
	}
	rootCmd.AddCommand(debugCmd)

	fetcherCmd := &cobra.Command{
		Use: "fetch",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := &Fetcher{
				URL:         args[0],
				PageLimit:   5,
				Concurrency: 10,
			}

			err := f.Run(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Printf("Item count: %d\n", len(f.Items))

			for _, item := range f.Items {
				fmt.Printf(
					"[%s] %s (%s)\n",
					item.GUID,
					item.Title,
					item.PublishDate,
				)
			}

			return nil
		},
	}

	pageCmd := &cobra.Command{
		Use: "page",
		RunE: func(cmd *cobra.Command, args []string) error {
			firstPage, err := NewListPage(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			b, err := json.MarshalIndent(firstPage, "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("%+v\n", string(b))

			return nil
		},
	}

	itemCmd := &cobra.Command{
		Use: "item",
		RunE: func(cmd *cobra.Command, args []string) error {
			page, err := NewItem(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			b, err := xml.MarshalIndent(page, "", "  ")
			if err != nil {
				return err
			}

			fmt.Printf("%+v\n", string(b))
			return nil
		},
	}

	debugCmd.AddCommand(
		fetcherCmd,
		pageCmd,
		itemCmd,
	)

	return rootCmd.ExecuteContext(ctx)
}
