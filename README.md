# Servicio de Ejecución de Código Remoto

Esta plataforma ha sido diseñada para ejecutar de forma segura y remota fragmentos de código enviados por los usuarios, siendo ideal para entornos de *coding challenges*. Utiliza contenedores Docker para aislar y ejecutar el código, garantizando que cada contenedor se destruya después de la ejecución. Además, está preparada para soportar múltiples usuarios concurrentes, escalando de manera eficiente según la demanda.

## Tabla de Contenidos

- [Introducción](#introducción)
- [Flujo de Ejecución del API](#flujo-de-ejecución-del-api)
  - [1. Recepción de la Solicitud](#1-recepción-de-la-solicitud)
  - [2. Encolado del Trabajo](#2-encolado-del-trabajo)
  - [3. Procesamiento por el Worker](#3-procesamiento-por-el-worker)
  - [4. Ejecución en el Contenedor Docker](#4-ejecución-en-el-contenedor-docker)
  - [5. Manejo y Almacenamiento de Resultados](#5-manejo-y-almacenamiento-de-resultados)
  - [6. Recuperación del Resultado](#6-recuperación-del-resultado)
- [Arquitectura de Red y Comunicación](#arquitectura-de-red-y-comunicación)
- [Ventajas del Enfoque HTTP](#ventajas-del-enfoque-http)
- [Consideraciones de Seguridad](#consideraciones-de-seguridad)

## Introducción

Esta API es el núcleo de un sistema de ejecución de código remoto, desarrollado para permitir a los usuarios enviar código en varios lenguajes (como Python, JavaScript, C++, entre otros) y obtener los resultados de forma segura. La ejecución se realiza en contenedores Docker aislados que se eliminan automáticamente tras el procesamiento, lo que garantiza que el sistema se mantenga limpio y seguro.

## Flujo de Ejecución del API

A continuación se detalla el flujo completo desde que se realiza la solicitud hasta que se obtiene el resultado:

### 1. Recepción de la Solicitud

- **Endpoint:** Se envía una solicitud `POST` a `/execute` junto con el código a ejecutar y el lenguaje seleccionado.
- **Validación:** El manejador de solicitudes `executeHandler` (definido en `api/main.go`) valida la petición.
- **Identificación del Trabajo:** Se genera un identificador único para el trabajo (Job ID) utilizando UUID, lo que permite rastrear cada ejecución de forma individual.

### 2. Encolado del Trabajo

- **Serialización:** El código, el lenguaje y el identificador se empaquetan en una estructura `Job`.
- **Cola de Redis:** La estructura se serializa a JSON y se empuja a una cola en Redis denominada `code_jobs`.
- **Respuesta Inmediata:** La API responde al cliente de forma inmediata, devolviendo el Job ID para que el usuario pueda posteriormente consultar el estado del proceso.

### 3. Procesamiento por el Worker

- **Polling de Redis:** Un servicio *worker* ejecutándose en `worker/main.go` monitorea la cola `code_jobs` utilizando el comando `BRPOP` para retirar trabajos de forma bloqueante.
- **Deserialización:** Al recibir un trabajo, el worker deserializa el JSON a un objeto `Job`.
- **Ejecución del Código:** Se invoca la función `executeCode`, encargada de gestionar el proceso de ejecución.

### 4. Ejecución en el Contenedor Docker

- **Almacenamiento Temporal del Código:** El worker guarda el código en una estructura en memoria (`codeStore`) asociada a un ID único.
- **Exposición vía HTTP:** Se dispone de un endpoint HTTP (`/code`) que sirve el código almacenado, permitiendo que el contenedor Docker lo recupere.
- **Lanzamiento del Contenedor:**
  - Se inicia un contenedor Docker que ejecuta el código en el lenguaje indicado.
  - Se utiliza una variable de entorno (`CODE_URL=http://worker:8081/code?id=...`) para que el contenedor sepa dónde obtener el código.
- **Ejecución y Captura de Salida:**
  - Dentro del contenedor, un script recupera el código mediante una petición HTTP al worker.
  - El código se guarda en un archivo temporal con la extensión correspondiente (.py, .js, .cpp, etc.).
  - El código se ejecuta con las herramientas específicas del lenguaje:
    - **Python:** Se ejecuta con el intérprete `python`.
    - **JavaScript:** Se ejecuta con Node.js.
    - **C++:** Se compila con `g++` y se ejecuta el binario resultante.
  - Se capturan la salida estándar y los errores generados durante la ejecución.
  - Los archivos temporales se eliminan tras la ejecución.

### 5. Manejo y Almacenamiento de Resultados

- **Captura del Resultado:** Una vez finalizada la ejecución, el contenedor devuelve los resultados (salida, errores y tiempos de ejecución) a la aplicación worker.
- **Construcción del Resultado:** El worker construye una estructura `JobResult` que incluye:
  - Estado de la ejecución.
  - Salida generada.
  - Mensajes de error (si existen).
  - Información de tiempo.
- **Almacenamiento en Redis:** El resultado se serializa a JSON y se almacena en Redis bajo la clave `result:{job_id}` con un tiempo de expiración de 24 horas.

### 6. Recuperación del Resultado

- **Consulta al Resultado:** El cliente puede realizar una solicitud `GET` a `/result/{job_id}` para obtener el resultado.
- **Manejo de Respuestas:**
  - Si el resultado existe, se devuelve el JSON con el estado, salida, errores, etc.
  - Si el trabajo aún está en proceso, se informa que el estado es "pending".
  - Si el Job ID no es válido o el trabajo no existe, se retorna un error.

## Arquitectura de Red y Comunicación

- **Red Interna Docker Compose:** Todos los servicios (API, Worker, Redis) se ejecutan dentro de una red definida en Docker Compose, facilitando la comunicación entre ellos.
- **Comunicación entre Contenedores:** El worker se comunica directamente con el demonio Docker mediante el socket, y los contenedores ejecutores alcanzan al worker usando el nombre del servicio "worker" en la red.
- **Enfoque HTTP:** La utilización de un endpoint HTTP para compartir el código evita problemas comunes relacionados con permisos en sistemas de archivos y montaje de volúmenes.

## Ventajas del Enfoque HTTP

El sistema adopta un enfoque basado en HTTP para la transferencia de código entre componentes, lo que ofrece varias ventajas:

1. **Eliminación de Problemas de Permisos:** Se evita el complicado manejo de permisos que puede surgir al montar volúmenes entre contenedores.
2. **Compatibilidad y Fiabilidad:** El método HTTP es ampliamente soportado y funciona de manera consistente en entornos Docker.
3. **Separación de Responsabilidades:** La lógica de envío y ejecución del código queda claramente separada, permitiendo una mayor modularidad y escalabilidad.
4. **Facilidad de Escalado:** El sistema es capaz de gestionar múltiples solicitudes concurrentes, gracias al manejo eficiente de colas en Redis y la naturaleza efímera de los contenedores.

## Consideraciones de Seguridad

- **Ejecución Aislada:** Cada fragmento de código se ejecuta en un contenedor Docker independiente, lo que minimiza el riesgo de afectaciones al sistema principal.
- **Destrucción de Contenedores:** Los contenedores son destruidos inmediatamente después de la ejecución, evitando persistencia de código potencialmente malicioso.
- **Validación de Entradas:** La API valida todas las solicitudes para prevenir inyecciones de código y otros vectores de ataque.
- **Monitoreo y Expiración de Resultados:** Los resultados se almacenan temporalmente y se eliminan automáticamente después de 24 horas para proteger la privacidad y seguridad de los datos.
