import { Form, Field as FormischField } from "@formisch/react";
import { Button } from "#/components/ui/button";
import { Field, FieldError, FieldLabel } from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { useSignupForm } from "../form/signup-form";

export const SignupForm = () => {
	const { signupForm, handleSubmit, error, isPending } = useSignupForm();
	return (
		<div className="h-full w-full">
			<div className="max-w-11/12"></div>
			<Form
				of={signupForm}
				id="rectangle-signup"
				onSubmit={handleSubmit}
				className="flex w-full flex-col gap-6 mt-8"
			>
				<FormischField of={signupForm} path={["name"]}>
					{(field) => (
						<Field data-invalid={field.errors !== null}>
							<FieldLabel htmlFor="name">Full Name</FieldLabel>
							<Input
								{...field.props}
								id="name"
								value={field.input ?? ""}
								aria-invalid={field.errors !== null}
								autoComplete="off"
							/>
							{field.errors && (
								<FieldError
									errors={field.errors.map((message) => ({ message }))}
								/>
							)}
						</Field>
					)}
				</FormischField>
				<FormischField of={signupForm} path={["email"]}>
					{(field) => (
						<Field data-invalid={field.errors !== null}>
							<FieldLabel htmlFor="email">Work Email</FieldLabel>
							<Input
								{...field.props}
								id="email"
								value={field.input ?? ""}
								aria-invalid={field.errors !== null}
								autoComplete="off"
								type="email"
							/>
							{field.errors && (
								<FieldError
									errors={field.errors.map((message) => ({ message }))}
								/>
							)}
						</Field>
					)}
				</FormischField>
				<FormischField of={signupForm} path={["password"]}>
					{(field) => (
						<Field data-invalid={field.errors !== null}>
							<FieldLabel htmlFor="password">Password</FieldLabel>
							<Input
								{...field.props}
								id="password"
								value={field.input ?? ""}
								aria-invalid={field.errors !== null}
								autoComplete="off"
								type="password"
							/>
							{field.errors && (
								<FieldError
									errors={field.errors.map((message) => ({ message }))}
								/>
							)}
						</Field>
					)}
				</FormischField>
				<FormischField of={signupForm} path={["confirmPassword"]}>
					{(field) => (
						<Field data-invalid={field.errors !== null}>
							<FieldLabel htmlFor="confirmPassword">
								Confirm Password
							</FieldLabel>
							<Input
								{...field.props}
								id="confirmPassword"
								value={field.input ?? ""}
								aria-invalid={field.errors !== null}
								autoComplete="off"
								type="password"
							/>
							{field.errors && (
								<FieldError
									errors={field.errors.map((message) => ({ message }))}
								/>
							)}
						</Field>
					)}
				</FormischField>
				{error && <p className="text-destructive text-sm">{error.message}</p>}
				<Button
					type="submit"
					form="rectangle-signup"
					className="mt-4"
					disabled={isPending}
				>
					{isPending ? "Creating Account..." : "Create Account"}
				</Button>
			</Form>
		</div>
	);
};
