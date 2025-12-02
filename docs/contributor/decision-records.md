# Decision Records

The following sections describe the main design decisions made.

## Handling of Rate Limiting Calling IAS API

We didn't implement any rate limit handling because the [Rate Limiting documentation of IAS](https://help.sap.com/docs/IDENTITY_AUTHENTICATION/6d6d63354d1242d185ab4830fc04feb1/e22ee47abf614565bcb29bb4ddbbf209.html) mentions the following: 

> To ensure a safe and stable environment, all requests have a limit of 50 concurrent requests per second. The requests are associated with the originating IP address, and not with the user making the requests.

Currently, we do not expect to exceed this rate limit as a reconciliation can perform a maximum of 5 sequential requests.  
There is also mention of a specific rate limit for SCIM endpoints, but we do not use these endpoints.

### Caching of Well-Known Token Endpoint

We read the known configuration of the IAS tenant that is used to create the applications to obtain the token endpoint. This token endpoint is then stored in the secret on the managed runtime along with the client ID and the client secret.

The assumption is that the token endpoint of the IAS tenant does not change without any notice of a breaking change.

To reduce the number of requests when creating an application client secret and thus increase the stability of the reconciliation, it was decided to cache the token endpoint on the first retrieval. The cached token endpoint is not invalidated during operator runtime; however, it is updated when the IAS credentials or tenant URL are changed.

### Referencing IAS Applications by Name

The IAS application is created with a name that matches the name of the EventingAuth CR. This name is the unique runtime ID of the cluster for which the IAS application is created.

Since we do not want to store the IAS application ID in the secret stored on the managed runtime, we can read the IAS application only by its name. During the creation of the application, existing applications with the same name are read. If an application with the same name exists, it is deleted, as we assume this is due to a failed reconciliation. If more than one application with the same name already exists, the reconciliation fails. The same behaviour occurs when reconciling the deletion of the EventingAuth CR.

It was decided not to delete any of the existing applications in this case, as it is an unexpected condition that may have been caused by manual actions, and we may want to keep the applications to find the cause of the issue.

### Handling of Failed IAS Application and Secret Creation

If the creation of the IAS application fails, the reconciliation is retried. If an application has already been created, it is deleted before creation is attempted again.

To avoid having multiple applications with the same name, the application is created again only if the deletion is successful.

During the application creation process, there are several steps that can fail. First, the application is created, then the client secret is created, and finally the client ID of the client secret is read.   

It was decided to always delete the application if any of these steps fail, as this makes the whole process more understandable and easier to maintain.  

The reason for this is that the existing application can only be reused if the reconciliation failed before the client secret was successfully created, as we have no way to retrieve the client secret the next time the reconciliation is performed. 

Additionally, if the creation of the secret on the managed runtime fails, we retrieve the created IAS application from memory instead of recreating it in the IAS. 