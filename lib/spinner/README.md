
	// 	var sp = spinner.New()
	// 	prompt := promptui.Prompt{
	// 		Label: "Number",
	// 	}

	// 	result, err := prompt.Run()
	// 	fmt.Println(err, result)

	// 	confirm := promptui.Prompt{
	// 		Label:     "Is every ok?",
	// 		IsConfirm: true,
	// 	}
	// 	result, err = confirm.Run()
	// 	fmt.Println(err, result)

	// 	// sp := spinner.New().Start()

	// 	stop1 := time.NewTimer(time.Second * 1)
	// 	stop2 := time.NewTimer(time.Second * 2)
	// 	stop3 := time.NewTimer(time.Second * 10)

	// OUT:
	// 	for {
	// 		select {
	// 		case <-stop1.C:
	// 			sp.SetMsg("Working on 1, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
	// 		case <-stop2.C:
	// 			sp.CheckPoint(icon.IconCheck, color.ColorBlue, "what", color.ColorPurple)
	// 			// sp.CheckPoint("O", spinner.ColorBlue, "what", spinner.ColorPurple)
	// 			sp.SetMsg("Working on 2, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
	// 			// sp.SetSpinGap(50 * time.Millisecond)
	// 		case <-stop3.C:
	// 			sp.Success("Everything good!")
	// 			sp.Success("Bye")
	// 			break OUT
	// 		}
	// 	}
