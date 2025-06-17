# sonarcloud-report

Este proyecto genera un reporte en CSV de la calidad de código de proyectos en SonarCloud y lo sube automáticamente a una página de Confluence. El proceso está preparado para ejecutarse como un Job en Google Cloud Run, y puede ser orquestado mediante Cloud Scheduler para su ejecución periódica.

## Despliegue en Cloud Run Job

1. **Construcción de la imagen Docker**

   ```sh
   docker build -t gcr.io/<PROJECT_ID>/sonarcloud-report:latest .
   docker push gcr.io/<PROJECT_ID>/sonarcloud-report:latest
   ```

2. **Creación del Job en Cloud Run**

   ```sh
   gcloud run jobs create sonarcloud-report-job \
     --image gcr.io/<PROJECT_ID>/sonarcloud-report:latest \
     --region <REGION> \
     --set-env-vars SONARCLOUD_ORG=... \
     --set-env-vars SONARCLOUD_TOKEN=... \
     --set-env-vars CONFLUENCE_PAGEID=... \
     --set-env-vars CONFLUENCE_ORG_URL=... \
     --set-env-vars CONFLUENCE_API_KEY=... \
     --set-env-vars CONFLUENCE_USERNAME=...
   ```

   > Reemplaza `<PROJECT_ID>` y `<REGION>` por los valores de tu proyecto de Google Cloud.

3. **Programar la ejecución con Cloud Scheduler**

   Crea un trigger HTTP para el Job y programa su ejecución:

   ```sh
   gcloud scheduler jobs create http sonarcloud-report-schedule \
     --schedule="0 7 * * *" \
     --uri=<TRIGGER_URL> \
     --http-method=POST \
     --oidc-service-account-email=<SERVICE_ACCOUNT>
   ```

   > Ajusta la expresión cron y los parámetros según tus necesidades.

## Variables de entorno requeridas

- `SONARCLOUD_ORG`: Organización de SonarCloud.
- `SONARCLOUD_TOKEN`: Token de acceso a SonarCloud.
- `CONFLUENCE_PAGEID`: ID de la página de Confluence donde se subirá el CSV.
- `CONFLUENCE_ORG_URL`: URL base de la organización en Confluence.
- `CONFLUENCE_API_KEY`: API Key de Confluence.
- `CONFLUENCE_USERNAME`: Usuario de Confluence.

## Notas

- El Job está preparado para ejecutarse de forma concurrente y tolerante a fallos de red.
- El CSV generado contiene métricas de calidad de código por proyecto y rama principal.
- El Job puede ser ejecutado manualmente desde Cloud Run o programado con Cloud Scheduler.

## Licencia

BSD-3-Clause

