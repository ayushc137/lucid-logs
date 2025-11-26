package docs

func init() {
	SwaggerInfo.Description = "Backend for the Daily Journal Application.\n\n" +
		"**Swagger Auth Quickstart**\n" +
		"1. Call `POST /api/v1/auth/login` with your credentials (admin@example.com / adminadmin).\n" +
		"2. Copy the `token` value from the response.\n" +
		"3. Click the **Authorize** button in Swagger UI and paste the token with the `Bearer ` prefix (for example: `Bearer eyJ...`).\n" +
		"4. Close the dialog; the token stays persisted while the tab is open."
}
