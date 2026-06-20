import { useQuery } from "@tanstack/react-query";
import { api } from "..";
import { API_ROUTES } from "../api-routes";
import { queries } from "../queries";

export type GetGithubOauthLinkOutput = {
	authUrl: string;
};

export const getGithubOauthLinkApi = async () => {
	return await api.request<GetGithubOauthLinkOutput>(API_ROUTES.auth.oauth.github);
};

export const useGetGithubOauthLinkApi = () => {
	const query = useQuery({
		queryKey: queries.auth.getGithubOauthLink.queryKey,
		queryFn: getGithubOauthLinkApi,
		enabled: false,
	});

	const { data, error, isFetching, refetch } = query;
	return {
		...query,
		data,
		error,
		isFetching,
		getGithubOauthLink: refetch,
	};
};
