import pykorm

@pykorm.k8s_custom_object('com.ie.ibm.hpsys', 'v1alpha1', 'datasets')
class Dataset(pykorm.NamespacedModel):
    local = pykorm.fields.Spec("local")
    remote = pykorm.fields.Spec("remote")